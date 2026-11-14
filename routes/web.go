package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/middleware"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-trading/app/accounts"
	"github.com/AlexGalhardo/vukem-trading/app/controllers/terminal"
	"github.com/AlexGalhardo/vukem-trading/resources"
)

func Web(r *router.Router, app *vukem.App) error {
	if err := vukem.DefaultMiddleware(app, r); err != nil {
		return err
	}
	r.UseServer(middleware.Gzip())
	r.UseServer(middleware.StaticFS(resources.Public(), middleware.StaticConfig{Prefix: "/"}))
	r.Use(app.Inertia.Middleware)
	// Financial and session data must never sit in a shared or browser cache; hashed assets are
	// served earlier by StaticFS and stay immutable.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "private, no-store")
			next.ServeHTTP(w, r)
		})
	})
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}
	h, err := terminal.New(app)
	if err != nil {
		return err
	}
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := web.NewCtx(w, r, web.ServicesFrom(r.Context()))
			inertia.Share(r.Context(), "appName", "vukem-trading")
			inertia.Share(r.Context(), "auth", func(context.Context) (any, error) {
				u, err := acc.Web.User(c)
				if err != nil {
					//nolint:nilerr // an anonymous visitor is not an error
					return map[string]any{"user": nil}, nil
				}
				return map[string]any{"user": map[string]any{"id": u.ID, "name": u.Name, "email": u.Email}}, nil
			})
			next.ServeHTTP(w, r)
		})
	})
	required := auth.Required(acc.Web)
	r.Get("/", h.Page("overview")).Name("home").Use(required)
	for _, section := range []string{"market", "strategies", "backtests", "orders", "positions", "risk", "audit", "settings", "recommendations"} {
		r.Get("/"+section, h.Page(section)).Use(required)
	}
	r.Get("/dashboard", func(c *web.Ctx) error { return c.Redirect("/") }).Use(required)
	limiters, err := app.RateLimiters()
	if err != nil {
		return err
	}
	// Online guessing of MCP_TOKEN is bounded per client address.
	r.Post("/mcp", h.MCP).Use(ratelimit.Middleware(limiters("mcp", 60, time.Minute), ratelimit.ByIP))
	r.Get("/ready", h.Ready)
	r.Get("/api/snapshot", h.Snapshot).Use(required)
	r.Get("/api/artifacts/{id}", h.Artifact).Use(required)
	r.Post("/api/datasets/import", h.Import).Use(required)
	r.Post("/api/datasets/fetch", h.Fetch).Use(required)
	r.Get("/api/datasets/{id}", h.Dataset).Use(required)
	r.Get("/api/strategy-kinds", h.Kinds).Use(required)
	r.Get("/api/recommendations/latest", h.Recommendations).Use(required)
	r.Get("/api/metrics", h.Metrics).Use(required)
	r.Get("/api/stream", h.Stream).Use(required)
	r.Get("/api/preferences", h.Preferences).Use(required)
	r.Put("/api/preferences", h.SavePreferences).Use(required)
	r.Post("/api/strategies", h.Strategy).Use(required)
	r.Post("/api/strategies/draft", h.StrategyDraft).Use(required)
	r.Post("/api/strategies/{id}/toggle", h.Toggle).Use(required)
	r.Post("/api/strategies/{id}/decision", h.Decision).Use(required)
	r.Post("/api/backtests", h.Backtest).Use(required)
	r.Get("/api/backtests/{id}", h.Job).Use(required)
	r.Post("/api/backtests/{id}/cancel", h.CancelJob).Use(required)
	r.Post("/api/accounts/{id}/policy", h.Policy).Use(required)
	r.Post("/api/accounts/{id}/ticket", h.Ticket).Use(required)
	r.Post("/api/accounts/{id}/control", h.Control).Use(required)
	r.Post("/api/accounts/{account}/orders/{id}/cancel", h.CancelOrder).Use(required)
	return nil
}
