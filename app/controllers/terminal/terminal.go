// Package terminal serves the trading terminal pages and its authenticated JSON API.
package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/sse"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-trading/internal/providers"
	"github.com/AlexGalhardo/vukem-trading/internal/reco"
	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

type Controller struct {
	Store trading.Store
	App   *vukem.App
}

func New(app *vukem.App) (*Controller, error) {
	if err := ValidateMCPToken(app.Source.Get("MCP_TOKEN")); err != nil {
		return nil, err
	}
	db, err := app.DB()
	return &Controller{trading.Store{DB: db}, app}, err
}

func decode(c *web.Ctx, out any) error {
	r := json.NewDecoder(io.LimitReader(c.Request().Body, 11<<20))
	r.DisallowUnknownFields()
	if err := r.Decode(out); err != nil {
		return errs.New(422, "invalid JSON or unknown fields")
	}
	var extra any
	if r.Decode(&extra) != io.EOF {
		return errs.New(422, "JSON must contain a single object")
	}
	return nil
}

func bad(err error) error {
	if err != nil {
		return errs.New(422, err.Error())
	}
	return nil
}
func accountID(id string) bool { return id == "b3" || id == "us" || id == "research" }

// accountFor names the account a market belongs to: replay, paper broker, or the research account for
// markets that only run backtests.
func accountFor(market string) string {
	switch market {
	case "B3":
		return "b3"
	case "US":
		return "us"
	}
	return "research"
}

func (h *Controller) Page(section string) func(*web.Ctx) error {
	return func(c *web.Ctx) error { return inertia.Render(c, "Terminal", inertia.Props{"section": section}) }
}

func (h *Controller) Snapshot(c *web.Ctx) error {
	ctx := c.Context()
	if err := h.Store.Initialize(ctx); err != nil {
		return err
	}
	accounts, err := h.Store.List(ctx, "trading_accounts", 10)
	if err != nil {
		return err
	}
	states := []map[string]any{}
	for _, r := range accounts {
		var a trading.State
		if err := json.Unmarshal(r.Payload, &a); err != nil {
			return err
		}
		orders := a.Orders
		if len(orders) > 100 {
			orders = orders[len(orders)-100:]
		}
		fills := a.Fills
		if len(fills) > 100 {
			fills = fills[len(fills)-100:]
		}
		dd := trading.Zero
		if a.Peak.IsPositive() {
			dd = a.Peak.Sub(a.Equity()).Div(a.Peak).Mul(trading.D("100"))
		}
		unrealized := trading.Zero
		for _, p := range a.Positions {
			unrealized = unrealized.Add(p.Mark.Sub(p.Average).Mul(p.Qty))
		}
		states = append(
			states,
			map[string]any{
				"id":            a.ID,
				"mode":          a.Mode,
				"currency":      a.Currency,
				"cash":          a.Cash,
				"equity":        a.Equity(),
				"realized":      a.Realized,
				"unrealized":    unrealized,
				"exposure":      a.Exposure(),
				"drawdown":      dd,
				"halted":        a.Halted,
				"reason":        a.Reason,
				"heartbeat":     a.Heartbeat,
				"reconciled_at": a.ReconciledAt,
				"orders":        orders,
				"positions":     a.Positions,
				"fills":         fills,
				"policy":        a.Policy,
				"cursors":       a.Cursors,
				"day_start":     a.DayStart,
				"peak":          a.Peak,
			},
		)
	}
	datasets, err := h.Store.List(ctx, "datasets", 100)
	if err != nil {
		return err
	}
	meta := []map[string]any{}
	for _, r := range datasets {
		var ds trading.Dataset
		if err := json.Unmarshal(r.Payload, &ds); err != nil {
			return err
		}
		var first, last time.Time
		lastClose := trading.Zero
		if len(ds.Candles) > 0 {
			first = ds.Candles[0].Time
			last = ds.Candles[len(ds.Candles)-1].Time
			lastClose = ds.Candles[len(ds.Candles)-1].Close
		}
		meta = append(
			meta,
			map[string]any{
				"id":         ds.ID,
				"symbol":     ds.Symbol,
				"currency":   ds.Currency,
				"source":     ds.Source,
				"hash":       ds.Hash,
				"license":    ds.License,
				"bars":       len(ds.Candles),
				"first":      first,
				"last":       last,
				"last_close": lastClose,
				"problems":   trading.DatasetProblems(ds),
			},
		)
	}
	strategies, err := h.Store.List(ctx, "strategies", 100)
	if err != nil {
		return err
	}
	jobs, err := h.Store.List(ctx, "backtests", 100)
	if err != nil {
		return err
	}
	for j := range jobs {
		var job trading.Job
		if err := json.Unmarshal(jobs[j].Payload, &job); err != nil {
			return err
		}
		if job.Result != nil {
			job.Result.Curve = nil
			job.Result.Fills = nil
			job.Result.Final = trading.State{}
		}
		jobs[j].Payload, _ = json.Marshal(job)
	}
	audit, err := h.Store.Audit(ctx)
	if err != nil {
		return err
	}
	return c.JSON(
		200,
		map[string]any{
			"accounts":    states,
			"datasets":    meta,
			"strategies":  strategies,
			"backtests":   jobs,
			"audit":       audit,
			"instruments": trading.Instruments,
			"coverage":    trading.CoverageMatrix(),
			"settings": map[string]any{
				"google_configured": h.App.Source.Get("SOCIAL_GOOGLE_CLIENT_ID") != "",
				"owner_configured":  h.App.Source.Get("OWNER_GOOGLE_EMAIL") != "",
				"alpaca_configured": h.App.Source.Get("ALPACA_API_KEY") != "" &&
					h.App.Source.Get("ALPACA_API_SECRET") != "",
				"live_trading":   trading.EvaluateLiveGate(h.App.Source.Get, "web").Enabled,
				"mode":           "paper_only",
				"feed":           "IEX",
				"timeframe":      5,
				"engine_version": "vukem-trading-1",
			},
			"server_time": time.Now().UTC(),
		},
	)
}

func (h *Controller) Import(c *web.Ctx) error {
	var in struct {
		Symbol  string `json:"symbol"`
		Source  string `json:"source"`
		License string `json:"license"`
		CSV     string `json:"csv"`
	}
	if err := decode(c, &in); err != nil {
		return err
	}
	ds, err := trading.ImportCSV(strings.NewReader(in.CSV), in.Symbol, in.Source, in.License)
	if err != nil {
		return bad(err)
	}
	if err := h.Store.Create(c.Context(), "datasets", ds.ID, "ready", "owner", ds); err != nil {
		return err
	}
	return c.JSON(201, map[string]any{"id": ds.ID, "bars": len(ds.Candles)})
}

func (h *Controller) Fetch(c *web.Ctx) error {
	var in struct {
		Symbol string `json:"symbol"`
		Days   int    `json:"days"`
	}
	if err := decode(c, &in); err != nil {
		return err
	}
	if in.Days < 1 || in.Days > 30 {
		return errs.New(422, "period must be between 1 and 30 days")
	}
	inst, err := trading.FindInstrument(in.Symbol)
	if err != nil {
		return errs.New(422, "Symbol outside the watchlist")
	}
	registry := providers.NewRegistry(h.App.Source.Get, providers.AlpacaIEX{}, providers.Binance{})
	provider, ok := registry.For(inst.Market)
	if !ok {
		return errs.New(422, "This market has no data provider; import a CSV instead")
	}
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()
	to := time.Now()
	if inst.Market == "US" {
		to = to.Add(-16 * time.Minute) // the free IEX feed lags by about 16 minutes
	}
	ds, err := registry.Fetch(ctx, provider.Name(), providers.Query{
		Symbol: in.Symbol, From: to.Add(-time.Duration(in.Days) * 24 * time.Hour), To: to,
	})
	if errors.Is(err, providers.ErrMissingCredential) {
		return errs.New(422, "Provider credentials are not configured on the server")
	}
	if err != nil {
		return errs.New(502, "Failed to fetch history; check the provider, its credentials and the period")
	}
	if err := h.Store.Create(c.Context(), "datasets", ds.ID, "ready", "owner", ds); err != nil {
		return err
	}
	return c.JSON(201, map[string]any{"id": ds.ID, "bars": len(ds.Candles)})
}

func (h *Controller) Dataset(c *web.Ctx) error {
	var ds trading.Dataset
	if err := h.Store.Get(c.Context(), "datasets", c.Param("id"), &ds); err != nil {
		return errs.New(404, "Dataset not found")
	}
	if len(ds.Candles) > 1000 {
		ds.Candles = ds.Candles[len(ds.Candles)-1000:]
	}
	return c.JSON(200, ds)
}

// Preferences returns the owner's saved workspace.
func (h *Controller) Preferences(c *web.Ctx) error {
	p, err := h.Store.Preferences(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(200, p)
}

// SavePreferences replaces the saved workspace after validating it.
func (h *Controller) SavePreferences(c *web.Ctx) error {
	var p trading.Preferences
	if err := decode(c, &p); err != nil {
		return err
	}
	if err := h.Store.SavePreferences(c.Context(), p); err != nil {
		return bad(err)
	}
	return c.JSON(200, p)
}

// Ticket places a paper order from one of the owner's templates, through the risk engine.
func (h *Controller) Ticket(c *web.Ctx) error {
	id := c.Param("id")
	if id != "b3" && id != "us" {
		return errs.New(404, "Orders can only be placed on a paper account")
	}
	var in struct {
		Key    string `json:"key"`
		Symbol string `json:"symbol"`
	}
	if err := decode(c, &in); err != nil {
		return err
	}
	inst, err := trading.FindInstrument(in.Symbol)
	if err != nil || accountFor(inst.Market) != id {
		return errs.New(422, "Symbol does not match the account")
	}
	prefs, err := h.Store.Preferences(c.Context())
	if err != nil {
		return err
	}
	var tmpl *trading.OrderTemplate
	for i := range prefs.Templates {
		if prefs.Templates[i].Key == in.Key {
			tmpl = &prefs.Templates[i]
		}
	}
	if tmpl == nil {
		return errs.New(422, "Unknown order template")
	}
	quotes, err := h.Store.LastQuotes(c.Context(), in.Symbol)
	if err != nil {
		return err
	}
	quote, ok := quotes[in.Symbol]
	if !ok {
		return errs.New(422, "No price for the symbol: import or fetch a dataset first")
	}
	price := quote.Price
	var placed trading.Order
	err = h.Store.Mutate(c.Context(), id, "owner", "ticket", func(a *trading.State) error {
		// B3 is a replay on a historical clock; only the broker account trades against the wall clock.
		if id == "us" && time.Since(quote.Time.Add(5*time.Minute)) > time.Duration(a.Policy.MaxAgeSeconds)*time.Second {
			return errors.New("the quote is older than the policy allows: fetch fresh data first")
		}
		o, err := a.Ticket(*tmpl, in.Symbol, price, "ticket/"+trading.ID(), time.Now().UTC())
		if err != nil {
			return err
		}
		placed = *o
		return nil
	})
	if err != nil {
		return bad(err)
	}
	return c.JSON(201, placed)
}

// Metrics returns the operational signals of every account.
func (h *Controller) Metrics(c *web.Ctx) error {
	records, err := h.Store.List(c.Context(), "trading_accounts", 10)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	out := make([]trading.OpsMetrics, 0, len(records))
	for _, r := range records {
		var a trading.State
		if err := json.Unmarshal(r.Payload, &a); err != nil {
			return err
		}
		out = append(out, trading.OperationalMetrics(a, now))
	}
	return c.JSON(200, map[string]any{"time": now, "accounts": out})
}

// Stream pushes a "changed" event whenever the data behind the terminal changes, so the page refetches
// at once instead of waiting for its next poll. Polling stays as the fallback.
func (h *Controller) Stream(c *web.Ctx) error {
	return sse.Serve(c.Writer(), c.Request(), func(s *sse.Stream) error {
		last := ""
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		for {
			signature, err := h.Store.ChangeSignature(s.Context())
			if err == nil && signature != last {
				last = signature
				if err := s.SendJSON("changed", map[string]string{"signature": signature}); err != nil {
					return nil // a write error means the client left
				}
			}
			select {
			case <-s.Context().Done():
				return nil
			case <-tick.C:
			}
		}
	})
}

// Decision records the owner's keep-or-disable verdict on a strategy, with the reason, in the audit trail.
func (h *Controller) Decision(c *web.Ctx) error {
	var in struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if err := decode(c, &in); err != nil {
		return err
	}
	if err := h.Store.RecordStrategyDecision(c.Context(), c.Param("id"), in.Decision, in.Note); err != nil {
		return bad(err)
	}
	return c.JSON(201, map[string]string{"status": "recorded"})
}

// Kinds serves the strategy kinds and their parameter schema, the single source for forms and validation.
func (h *Controller) Kinds(c *web.Ctx) error {
	return c.JSON(200, trading.Kinds())
}

func (h *Controller) Strategy(c *web.Ctx) error { return h.createStrategy(c, false) }

// StrategyDraft stores a proposed configuration, validated like any other, as a draft that stays disabled
// until the owner approves it. Use it for configurations written outside the owner's session.
func (h *Controller) StrategyDraft(c *web.Ctx) error { return h.createStrategy(c, true) }

func (h *Controller) createStrategy(c *web.Ctx, draft bool) error {
	var s trading.Strategy
	if err := decode(c, &s); err != nil {
		return err
	}
	s.Draft = draft
	if err := s.Validate(); err != nil {
		return bad(err)
	}
	if len(strings.TrimSpace(s.Name)) < 3 || len(s.Name) > 80 || !accountID(s.AccountID) {
		return errs.New(422, "Invalid name or account")
	}
	inst, _ := trading.FindInstrument(s.Symbol)
	if s.AccountID != accountFor(inst.Market) {
		return errs.New(422, "Symbol does not match the account")
	}
	if s.AccountID != "us" {
		var ds trading.Dataset
		if err := h.Store.Get(
			c.Context(),
			"datasets",
			s.DatasetID,
			&ds,
		); err != nil || ds.Symbol != s.Symbol ||
			len(ds.Candles) < s.Slow+2 {
			return errs.New(422, "Select a dataset compatible with warmup")
		}
	}
	s.ID = trading.ID()
	s.Version = 1
	s.Enabled = false
	if err := h.Store.CreateStrategy(c.Context(), &s); err != nil {
		return err
	}
	return c.JSON(201, s)
}

func (h *Controller) Toggle(c *web.Ctx) error {
	var in struct {
		Enabled      bool `json:"enabled"`
		ApproveDraft bool `json:"approve_draft"`
	}
	if err := decode(c, &in); err != nil {
		return err
	}
	var s trading.Strategy
	if err := h.Store.Get(c.Context(), "strategies", c.Param("id"), &s); err != nil {
		return errs.New(404, "Strategy not found")
	}
	if in.Enabled && s.Draft {
		if !in.ApproveDraft {
			return errs.New(422, "This strategy is a draft: review it, then approve it explicitly")
		}
		if err := h.Store.ApproveDraft(c.Context(), s.ID); err != nil {
			return err
		}
	}
	if in.Enabled && s.AccountID == "research" {
		return errs.New(422, "Research strategies run in backtests only")
	}
	if !in.Enabled {
		var a trading.State
		if err := h.Store.Get(c.Context(), "trading_accounts", s.AccountID, &a); err != nil {
			return err
		}
		if p := a.Positions[s.Symbol]; p.Qty.IsPositive() {
			return errs.New(
				422,
				"There is a position on the symbol. Block entries and keep the strategy until it is closed.",
			)
		}
		for _, o := range a.Orders {
			if o.Symbol == s.Symbol && o.Open() {
				return errs.New(422, "There are pending orders on the symbol")
			}
		}
	}
	status := "disabled"
	if in.Enabled {
		status = "enabled"
	}
	if err := h.Store.ToggleStrategy(c.Context(), s.ID, in.Enabled); err != nil {
		return bad(err)
	}
	return c.JSON(200, map[string]string{"status": status})
}

func (h *Controller) Backtest(c *web.Ctx) error {
	var in struct {
		StrategyID string `json:"strategy_id"`
		DatasetID  string `json:"dataset_id"`
		Partition  string `json:"partition"`
		// RepeatValidation acknowledges that validation was already used on this strategy version.
		RepeatValidation bool `json:"repeat_validation"`
	}
	if err := decode(c, &in); err != nil {
		return err
	}
	if !slices.Contains([]string{"train", "test", "validation", "all", "walkforward"}, in.Partition) {
		return errs.New(422, "Invalid partition")
	}
	var s trading.Strategy
	var ds trading.Dataset
	if err := h.Store.Get(c.Context(), "strategies", in.StrategyID, &s); err != nil {
		return errs.New(422, "Invalid strategy")
	}
	if err := h.Store.Get(c.Context(), "datasets", in.DatasetID, &ds); err != nil || ds.Symbol != s.Symbol {
		return errs.New(422, "Incompatible dataset")
	}
	if in.Partition == "validation" && !in.RepeatValidation {
		used, err := h.Store.ValidationUsed(c.Context(), s.ID)
		if err != nil {
			return err
		}
		if used {
			return errs.New(409, "Validation was already used on this strategy version; create a new version instead")
		}
	}
	var a trading.State
	if err := h.Store.Get(c.Context(), "trading_accounts", s.AccountID, &a); err != nil {
		return err
	}
	job := trading.Job{
		ID:          trading.ID(),
		DatasetID:   ds.ID,
		Strategy:    s,
		Policy:      a.Policy,
		Partition:   in.Partition,
		CreatedAt:   time.Now().UTC(),
		CodeVersion: trading.BuildVersion,
	}
	if err := h.Store.Create(c.Context(), "backtests", job.ID, "queued", "owner", job); err != nil {
		return err
	}
	return c.JSON(202, job)
}

func (h *Controller) Job(c *web.Ctx) error {
	var job trading.Job
	if err := h.Store.Get(c.Context(), "backtests", c.Param("id"), &job); err != nil {
		return errs.New(404, "Backtest not found")
	}
	if job.Result != nil {
		var result trading.Result
		if err := h.Store.Get(c.Context(), "artifacts", job.ID, &result); err != nil {
			return err
		}
		job.Result = &result
	}
	return c.JSON(200, job)
}

func (h *Controller) CancelJob(c *web.Ctx) error {
	if err := h.Store.Status(c.Context(), "backtests", c.Param("id"), "canceled", "owner"); err != nil {
		return errs.New(422, "Backtest already finished or does not exist")
	}
	return c.JSON(200, map[string]string{"status": "canceled"})
}

func (h *Controller) Policy(c *web.Ctx) error {
	id := c.Param("id")
	if !accountID(id) {
		return errs.New(404, "Invalid account")
	}
	var p trading.Policy
	if err := decode(c, &p); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return bad(err)
	}
	err := h.Store.Mutate(c.Context(), id, "owner", "risk.policy", func(a *trading.State) error {
		if !a.Halted {
			return errors.New("block the account before changing limits")
		}
		a.Policy = p
		a.Generation++
		return nil
	})
	if err != nil {
		return bad(err)
	}
	return c.JSON(200, p)
}

func (h *Controller) Control(c *web.Ctx) error {
	id := c.Param("id")
	if !accountID(id) {
		return errs.New(404, "Invalid account")
	}
	var in struct {
		Action  string `json:"action"`
		Confirm string `json:"confirm"`
	}
	if err := decode(c, &in); err != nil {
		return err
	}
	if in.Action == "reset" {
		if id != "b3" || in.Confirm != "RESET" {
			return errs.New(422, "reset allowed only on B3 replay with confirmation")
		}
		if err := h.Store.ResetReplay(c.Context()); err != nil {
			return bad(err)
		}
		return c.JSON(200, map[string]string{"status": "archived_and_reset"})
	}
	err := h.Store.Mutate(c.Context(), id, "owner", "control."+in.Action, func(a *trading.State) error {
		switch in.Action {
		case "halt":
			a.Halt("Block requested by the owner")
		case "resume":
			if in.Confirm != "PAPER" {
				return errors.New("PAPER confirmation required")
			}
			if a.Heartbeat.IsZero() || time.Since(a.Heartbeat) > 45*time.Second {
				return errors.New("worker is not healthy")
			}
			for _, o := range a.Orders {
				if o.Status == "unknown" || o.Status == "dispatching" || o.Status == "cancel_pending" {
					return errors.New("there are orders without a final state")
				}
			}
			if id == "us" && (a.ReconciledAt.IsZero() || time.Since(a.ReconciledAt) > 30*time.Second) {
				return errors.New("recent reconciliation required")
			}
			if a.DayStart.IsPositive() &&
				a.Equity().
					LessThanOrEqual(a.DayStart.Mul(trading.D("1").Sub(trading.Fraction(a.Policy.DailyLossBPS)))) {
				return errors.New("daily loss limit reached")
			}
			a.Halted = false
			a.Reason = ""
			a.Generation++
		case "cancel":
			a.Halt("Cancellation requested")
			for j := range a.Orders {
				o := &a.Orders[j]
				if o.Open() {
					if o.Status == "approved" || id == "b3" {
						o.Status = "canceled"
						o.Reserved = trading.Zero
					} else {
						o.Status = "cancel_pending"
					}
				}
			}
		case "flatten":
			if in.Confirm != "ZERAR" {
				return errors.New("ZERAR confirmation required")
			}
			a.Halt("Flatten requested; monitor orders and positions")
			if a.ReconciledAt.IsZero() || time.Since(a.ReconciledAt) > 30*time.Second {
				return errors.New("recent reconciliation required; block and reconcile first")
			}
			for symbol, p := range a.Positions {
				s := trading.Strategy{Symbol: symbol}
				o, err := a.Reserve(s, "sell", trading.ID()+"/flatten", p.Mark, time.Now().UTC())
				if err != nil {
					return err
				}
				if id == "b3" {
					return errors.New("B3 replay: close through the strategy to preserve the historical clock")
				}
				_ = o
			}
		default:
			return errors.New("invalid action")
		}
		return nil
	})
	if err != nil {
		return bad(err)
	}
	return c.JSON(200, map[string]string{"status": "recorded"})
}

func (h *Controller) CancelOrder(c *web.Ctx) error {
	id := c.Param("account")
	if !accountID(id) {
		return errs.New(404, "Invalid account")
	}
	err := h.Store.Mutate(c.Context(), id, "owner", "order.cancel", func(a *trading.State) error {
		o := a.Order(c.Param("id"))
		if o == nil {
			return errors.New("order not found")
		}
		if o.Open() {
			if o.Status == "approved" || id == "b3" {
				o.Status = "canceled"
				o.Reserved = trading.Zero
			} else {
				o.Status = "cancel_pending"
			}
		}
		return nil
	})
	if err != nil {
		return bad(err)
	}
	return c.JSON(200, map[string]string{"status": "recorded"})
}

func (h *Controller) Ready(c *web.Ctx) error {
	if err := h.Store.DB.SQL().PingContext(c.Context()); err != nil {
		return c.JSON(503, map[string]string{"status": "unavailable"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Controller) Artifact(c *web.Ctx) error {
	var raw json.RawMessage
	if err := h.Store.Get(c.Context(), "artifacts", c.Param("id"), &raw); err != nil {
		return errs.New(404, "Artifact not found")
	}
	return c.JSON(200, raw)
}

// Recommendations returns the newest daily ranking, or an empty report that still carries the disclaimer
// when the job has not run yet.
func (h *Controller) Recommendations(c *web.Ctx) error {
	db, err := h.App.DB()
	if err != nil {
		return err
	}
	rep, ok, err := reco.Store{DB: db}.Latest(c.Context())
	if err != nil {
		return err
	}
	if !ok {
		rep = reco.Report{
			Disclaimer: reco.Disclaimer, Criterion: reco.Criterion, Strategy: reco.StrategyDescription,
			B3: reco.B3Note, Markets: []reco.MarketReport{},
		}
	}
	return c.JSON(200, rep)
}
