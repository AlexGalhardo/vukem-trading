package tests

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-trading/app/commands"
	"github.com/AlexGalhardo/vukem-trading/bootstrap"
	"github.com/AlexGalhardo/vukem-trading/internal/engine"
	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

func terminalApp(t *testing.T, id, email string, verified bool) *vukemtest.App {
	t.Helper()
	v := "false"
	if verified {
		v = "true"
	}
	return vukemtest.New(
		t,
		bootstrap.New,
		vukemtest.Options{
			Env: map[string]string{
				"OWNER_GOOGLE_EMAIL":         "owner@example.com",
				"SOCIAL_FAKE_ID":             id,
				"SOCIAL_FAKE_EMAIL":          email,
				"SOCIAL_FAKE_EMAIL_VERIFIED": v,
			},
		},
	)
}

func signIn(t *testing.T, a *vukemtest.App) *vukemtest.Client {
	t.Helper()
	c := a.Client(t)
	r := c.Get("/auth/google/redirect")
	c.Get(r.Header("Location")).AssertRedirect("/")
	return c
}

func TestPrivateRoutesAndGoogleOnly(t *testing.T) {
	t.Parallel()
	a := terminalApp(t, "owner-1", "owner@example.com", true)
	c := a.Client(t)
	c.Get("/").AssertRedirect("/login")
	c.Get("/api/snapshot").AssertRedirect("/login")
	c.GetJSON("/api/snapshot").AssertUnauthorized()
	for _, path := range []string{"/register", "/forgot-password", "/posts"} {
		c.Get(path).AssertNotFound()
	}
	c.Get("/login").AssertOK().AssertInertia("Auth/Login")
	c.PostJSON("/login", map[string]any{}).AssertStatus(405)
	signIn(t, a).Get("/").AssertOK().AssertInertia("Terminal")
}

func TestGoogleGateAndReplay(t *testing.T) {
	for _, tc := range []struct {
		name, id, email string
		verified        bool
	}{{"foreign", "foreign", "intruder@example.com", true}, {"unverified", "owner", "owner@example.com", false}} {
		t.Run(tc.name, func(t *testing.T) {
			a := terminalApp(t, tc.id, tc.email, tc.verified)
			c := a.Client(t)
			r := c.Get("/auth/google/redirect")
			c.Get(r.Header("Location")).AssertForbidden()
			c.Get("/").AssertRedirect("/login")
			vukemtest.AssertDatabaseCount(t, a, "users", 0)
		})
	}
	a := terminalApp(t, "owner", "owner@example.com", true)
	c := a.Client(t)
	r := c.Get("/auth/google/redirect")
	c.Get(r.Header("Location")).AssertRedirect("/")
	c.Get(r.Header("Location")).AssertStatus(400)
	c.PostJSON("/logout", map[string]any{}).AssertRedirect("/login")
	c.Get("/").AssertRedirect("/login")
}

func TestTerminalPersistenceAndValidation(t *testing.T) {
	a := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, a).AcceptJSON()
	c.GetJSON("/api/snapshot").
		AssertOK().
		AssertJSONCount("accounts", 3).
		AssertJSONCount("coverage", 4).
		AssertJSONPath("settings.mode", "paper_only")
	db, err := a.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := trading.Store{DB: db}
	c.PostJSON("/api/datasets/import", map[string]any{"symbol": "PETR4", "source": "licensed fixture", "license": "internal test", "csv": trading.SampleCSV()}).
		AssertCreated()
	vukemtest.AssertDatabaseCount(t, a, "datasets", 1)
	c.PostJSON("/api/datasets/import", map[string]any{"symbol": "PETR4", "source": "licensed fixture", "license": "test", "csv": "time,open,high,low,close,volume\n2026-10-26T10:00:00-03:00,NaN,30,29,30,100\n"}).
		AssertUnprocessable()
	c.PostJSON("/api/accounts/b3/control", map[string]any{"action": "resume", "confirm": "PAPER"}).AssertUnprocessable()
	c.PostJSON("/api/accounts/b3/control", map[string]any{"action": "halt"}).AssertOK()
	var state trading.State
	if err := store.Get(t.Context(), "trading_accounts", "b3", &state); err != nil {
		t.Fatal(err)
	}
	if !state.Halted || state.Generation != 1 {
		t.Fatalf("halt not persisted: %+v", state)
	}
	c.PostJSON("/api/accounts/foreign/control", map[string]any{"action": "halt"}).AssertNotFound()
	c.PostJSON("/api/accounts/b3/policy", map[string]any{"risk_bps": 90000}).AssertUnprocessable()
	c.PostJSON("/api/backtests", map[string]any{"strategy_id": "missing", "dataset_id": "missing", "partition": "test"}).
		AssertUnprocessable()
}

func TestOwnerRecoveryRevokesSessions(t *testing.T) {
	a := terminalApp(t, "owner-stable", "owner@example.com", true)
	c := signIn(t, a).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	command := &commands.OwnerReset{}
	fs := flag.NewFlagSet("recovery", flag.ContinueOnError)
	command.Flags(fs)
	if err := command.Run(t.Context(), a.App, nil); err == nil {
		t.Fatal("recovery without confirmation accepted")
	}
	if err := fs.Parse([]string{"--confirm", "RESET-OWNER", "--reason", "Owner recovery test"}); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(t.Context(), a.App, nil); err != nil {
		t.Fatal(err)
	}
	c.GetJSON("/api/snapshot").AssertUnauthorized()
	signIn(t, a).Get("/").AssertOK()
	vukemtest.AssertDatabaseCount(t, a, "users", 1)
}

func TestStrategyVersionsActivationAndReplayArchive(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	input := map[string]any{
		"name":       "Versioned test",
		"kind":       "ma_cross",
		"symbol":     "AAPL",
		"account_id": "us",
		"fast":       2,
		"slow":       4,
		"threshold":  2,
		"stop_bps":   100,
		"target_bps": 200,
	}
	first := c.PostJSON("/api/strategies", input).AssertCreated().AssertJSONPath("version", 1)
	id, _ := first.JSONPath("id")
	second := c.PostJSON("/api/strategies", input).AssertCreated().AssertJSONPath("version", 2)
	secondID, _ := second.JSONPath("id")
	c.PostJSON("/api/strategies/"+id.(string)+"/toggle", map[string]any{"enabled": true}).AssertOK()
	c.PostJSON("/api/strategies/"+secondID.(string)+"/toggle", map[string]any{"enabled": true}).AssertUnprocessable()
	c.PostJSON("/api/accounts/b3/control", map[string]any{"action": "reset", "confirm": "RESET"}).AssertOK()
	vukemtest.AssertDatabaseCount(t, app, "artifacts", 1)
	c.PostJSON("/api/accounts/us/control", map[string]any{"action": "reset", "confirm": "RESET"}).AssertUnprocessable()
}

func TestStrategyKindsSchemaIsServedAndEnforced(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	c.GetJSON("/api/strategy-kinds").AssertOK().AssertJSONPath("0.kind", "ma_cross")
	base := map[string]any{
		"name": "Schema test", "kind": "ma_cross", "symbol": "AAPL", "account_id": "us",
		"fast": 2, "slow": 4, "threshold": 2, "stop_bps": 100, "target_bps": 200,
	}
	for field, bad := range map[string]any{"stop_bps": 9, "fast": 1, "slow": 501, "threshold": 0.05, "target_bps": 5001} {
		input := maps.Clone(base)
		input[field] = bad
		c.PostJSON("/api/strategies", input).AssertUnprocessable()
	}
	c.PostJSON("/api/strategies", base).AssertCreated()
}

func TestValidationPartitionIsUsedOncePerStrategyVersion(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	var rows strings.Builder
	rows.WriteString("time,open,high,low,close,volume\n")
	price := 30.0
	start := time.Date(2026, 10, 26, 10, 0, 0, 0, time.FixedZone("BRT", -3*3600))
	for i := range 60 {
		next := price + []float64{0.1, -0.05, 0.08, -0.12}[i%4]
		fmt.Fprintf(
			&rows,
			"%s,%.2f,%.2f,%.2f,%.2f,100000\n",
			start.Add(time.Duration(i)*5*time.Minute).Format(time.RFC3339),
			price,
			max(price, next)+0.05,
			min(price, next)-0.05,
			next,
		)
		price = next
	}
	imported := c.PostJSON("/api/datasets/import", map[string]any{
		"symbol": "PETR4", "source": "fixture", "license": "internal test", "csv": rows.String(),
	}).AssertCreated()
	datasetID, _ := imported.JSONPath("id")
	created := c.PostJSON("/api/strategies", map[string]any{
		"name": "Validation once", "kind": "ma_cross", "symbol": "PETR4", "account_id": "b3", "dataset_id": datasetID,
		"fast": 2, "slow": 5, "threshold": 2, "stop_bps": 100, "target_bps": 200,
	}).AssertCreated()
	strategyID, _ := created.JSONPath("id")
	run := func(partition string, extra map[string]any) *vukemtest.Response {
		in := map[string]any{"strategy_id": strategyID, "dataset_id": datasetID, "partition": partition}
		maps.Copy(in, extra)
		return c.PostJSON("/api/backtests", in)
	}
	run("test", nil).AssertStatus(202)
	run("test", nil).AssertStatus(202)
	run("validation", nil).AssertStatus(202)
	run("validation", nil).AssertStatus(409)
	run("validation", map[string]any{"repeat_validation": true}).AssertStatus(202)
	run("walkforward", nil).AssertStatus(202)
	run("holdout", nil).AssertUnprocessable()
}

func TestResearchStrategiesNeverGoLive(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	var rows strings.Builder
	rows.WriteString("time,open,high,low,close,volume\n")
	start := time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	for i := range 40 {
		p := 60000 + float64(i%5)*10
		fmt.Fprintf(
			&rows,
			"%s,%.2f,%.2f,%.2f,%.2f,250\n",
			start.Add(time.Duration(i)*5*time.Minute).Format(time.RFC3339),
			p,
			p+20,
			p-20,
			p+5,
		)
	}
	imported := c.PostJSON("/api/datasets/import", map[string]any{
		"symbol": "BTCUSDT", "source": "fixture", "license": "internal test", "csv": rows.String(),
	}).AssertCreated()
	datasetID, _ := imported.JSONPath("id")
	strategy := map[string]any{
		"name": "Crypto research", "kind": "ma_cross", "symbol": "BTCUSDT", "account_id": "research",
		"dataset_id": datasetID, "fast": 2, "slow": 5, "threshold": 2, "stop_bps": 100, "target_bps": 200,
	}
	wrong := maps.Clone(strategy)
	wrong["account_id"] = "b3"
	c.PostJSON("/api/strategies", wrong).AssertUnprocessable()
	created := c.PostJSON("/api/strategies", strategy).AssertCreated()
	id, _ := created.JSONPath("id")
	c.PostJSON("/api/strategies/"+id.(string)+"/toggle", map[string]any{"enabled": true}).AssertUnprocessable()
	c.PostJSON("/api/backtests", map[string]any{"strategy_id": id, "dataset_id": datasetID, "partition": "all"}).
		AssertStatus(202)
}

func TestFetchRefusesMarketsWithoutAProvider(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	c.PostJSON("/api/datasets/fetch", map[string]any{"symbol": "PETR4", "days": 5}).AssertUnprocessable()
	c.PostJSON("/api/datasets/fetch", map[string]any{"symbol": "AAPL", "days": 5}).AssertUnprocessable()
}

func TestPreferencesPersistAndAreValidated(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/preferences").AssertOK().AssertJSONPath("account", "b3")
	prefs := trading.DefaultPreferences()
	prefs.Account, prefs.Symbol = "us", "AAPL"
	prefs.Watchlist = []string{"AAPL", "NVDA", "BTCUSDT"}
	c.PutJSON("/api/preferences", prefs).AssertOK()
	c.GetJSON("/api/preferences").AssertOK().AssertJSONPath("account", "us").AssertJSONCount("watchlist", 3)
	bad := prefs
	bad.Watchlist = []string{"AAPL", "NOPE"}
	c.PutJSON("/api/preferences", bad).AssertUnprocessable()
	c.GetJSON("/api/preferences").AssertOK().AssertJSONCount("watchlist", 3)
}

func TestEngineFiresAPriceAlertOnceAndPersistsIt(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := trading.Store{DB: db}
	ctx := t.Context()
	at := time.Date(2026, 11, 4, 15, 0, 0, 0, time.UTC)
	candles := []trading.Candle{
		{
			Time:   at,
			Open:   trading.D("100"),
			High:   trading.D("101"),
			Low:    trading.D("99"),
			Close:  trading.D("100"),
			Volume: trading.D("1000"),
		},
		{
			Time:   at.Add(5 * time.Minute),
			Open:   trading.D("100"),
			High:   trading.D("212"),
			Low:    trading.D("99"),
			Close:  trading.D("210"),
			Volume: trading.D("1000"),
		},
	}
	ds := trading.Dataset{ID: "alerts-ds", Symbol: "AAPL", Candles: candles, CreatedAt: at}
	if err := store.Create(ctx, "datasets", ds.ID, "ready", "test", ds); err != nil {
		t.Fatal(err)
	}
	prefs := trading.DefaultPreferences()
	prefs.Alerts = []trading.PriceAlert{
		{ID: "up", Symbol: "AAPL", Direction: "above", Price: trading.D("200"), Active: true},
		{ID: "never", Symbol: "AAPL", Direction: "below", Price: trading.D("50"), Active: true},
	}
	if err := store.SavePreferences(ctx, prefs); err != nil {
		t.Fatal(err)
	}
	var sent []string
	e := engine.Engine{
		Store: store,
		Log:   slog.Default(),
		Alert: func(_ context.Context, msg string) { sent = append(sent, msg) },
	}
	e.CheckAlerts(ctx)
	e.CheckAlerts(ctx)
	if len(sent) != 1 || !strings.Contains(sent[0], "AAPL is above 200") {
		t.Fatalf("expected exactly one alert, got %v", sent)
	}
	saved, err := store.Preferences(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Alerts[0].Active || saved.Alerts[0].TriggeredAt.IsZero() || !saved.Alerts[1].Active {
		t.Fatalf("the fired alert must be stored as inactive and the other left alone: %+v", saved.Alerts)
	}
}

func TestTicketPlacesAPaperOrderThroughTheRiskEngine(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := trading.Store{DB: db}
	c.PostJSON("/api/datasets/import", map[string]any{
		"symbol": "PETR4", "source": "fixture", "license": "internal test", "csv": trading.SampleCSV(),
	}).AssertCreated()
	body := map[string]any{"key": "b", "symbol": "PETR4"}
	c.PostJSON("/api/accounts/b3/ticket", body).AssertUnprocessable() // the account starts blocked
	if err := store.Mutate(t.Context(), "b3", "test", "arm", func(a *trading.State) error {
		a.Halted, a.Reason = false, ""
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	placed := c.PostJSON("/api/accounts/b3/ticket", body).AssertCreated().AssertJSONPath("status", "approved")
	if side, _ := placed.JSONPath("side"); side != "buy" {
		t.Fatalf("the template buys, got %v", side)
	}
	c.PostJSON("/api/accounts/b3/ticket", map[string]any{"key": "nope", "symbol": "PETR4"}).AssertUnprocessable()
	c.PostJSON("/api/accounts/b3/ticket", map[string]any{"key": "b", "symbol": "AAPL"}).AssertUnprocessable()
	c.PostJSON("/api/accounts/research/ticket", body).AssertNotFound()
	var state trading.State
	if err := store.Get(t.Context(), "trading_accounts", "b3", &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Orders) != 1 || !state.Exposure().IsPositive() {
		t.Fatalf("exactly one reserved order expected, got %d", len(state.Orders))
	}
}

func TestMetricsEndpointIsAuthenticated(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	anonymous := app.Client(t)
	anonymous.GetJSON("/api/metrics").AssertUnauthorized()
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	c.GetJSON("/api/metrics").AssertOK().AssertJSONCount("accounts", 3).AssertJSONPath("accounts.0.halted", true)
}

func TestDraftStrategyNeedsExplicitApprovalToEnable(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	proposal := map[string]any{
		"name": "Proposed by an assistant", "kind": "opening_range", "symbol": "AAPL", "account_id": "us",
		"fast": 6, "slow": 12, "threshold": 2, "stop_bps": 100, "target_bps": 200, "draft": false,
	}
	invalid := maps.Clone(proposal)
	invalid["stop_bps"] = 1
	c.PostJSON("/api/strategies/draft", invalid).AssertUnprocessable()
	created := c.PostJSON("/api/strategies/draft", proposal).AssertCreated().AssertJSONPath("draft", true)
	id, _ := created.JSONPath("id")
	toggle := "/api/strategies/" + id.(string) + "/toggle"
	c.PostJSON(toggle, map[string]any{"enabled": true}).AssertUnprocessable()
	c.PostJSON(toggle, map[string]any{"enabled": true, "approve_draft": false}).AssertUnprocessable()
	c.PostJSON(toggle, map[string]any{"enabled": true, "approve_draft": true}).AssertOK()
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}
	var st trading.Strategy
	if err := (trading.Store{DB: db}).Get(t.Context(), "strategies", id.(string), &st); err != nil {
		t.Fatal(err)
	}
	if st.Draft {
		t.Fatal("approval must clear the draft flag")
	}
	// A strategy created normally is never a draft, whatever the request says.
	normal := maps.Clone(proposal)
	normal["name"] = "Created by the owner"
	normal["draft"] = true
	c.PostJSON("/api/strategies", normal).AssertCreated().AssertJSONPath("draft", false)
}

func TestChangeSignatureMovesWhenSomethingIsWritten(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := trading.Store{DB: db}
	before, err := store.ChangeSignature(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	again, _ := store.ChangeSignature(t.Context())
	if before != again {
		t.Fatal("the signature must be stable while nothing changes")
	}
	c.PostJSON("/api/datasets/import", map[string]any{
		"symbol": "PETR4", "source": "fixture", "license": "internal test", "csv": trading.SampleCSV(),
	}).AssertCreated()
	after, _ := store.ChangeSignature(t.Context())
	if after == before {
		t.Fatal("importing a dataset must change the signature")
	}
	app.Client(t).GetJSON("/api/stream").AssertUnauthorized()
}

func TestStrategyDecisionIsRecordedInTheAuditTrail(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	created := c.PostJSON("/api/strategies", map[string]any{
		"name": "Decision test", "kind": "ma_cross", "symbol": "AAPL", "account_id": "us",
		"fast": 2, "slow": 4, "threshold": 2, "stop_bps": 100, "target_bps": 200,
	}).AssertCreated()
	id, _ := created.JSONPath("id")
	path := "/api/strategies/" + id.(string) + "/decision"
	c.PostJSON(path, map[string]any{"decision": "maybe", "note": "not a verdict at all"}).AssertUnprocessable()
	c.PostJSON(path, map[string]any{"decision": "keep", "note": "short"}).AssertUnprocessable()
	c.PostJSON("/api/strategies/missing/decision", map[string]any{"decision": "keep", "note": "a long enough reason"}).
		AssertUnprocessable()
	c.PostJSON(path, map[string]any{"decision": "disable", "note": "paper drawdown exceeded the backtest's worst case"}).
		AssertCreated()
	c.GetJSON("/api/snapshot").AssertOK().AssertJSONPath("audit.0.action", "strategy.decision")
}

func TestGoogleRedirectUsesTheCanonicalAppURL(t *testing.T) {
	t.Parallel()
	a := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{
		"APP_URL":                 "https://app.trading.vukem.dev",
		"OWNER_GOOGLE_EMAIL":      "owner@example.com",
		"SOCIAL_GOOGLE_CLIENT_ID": "client-id",
	}})
	loc := a.Client(t).Get("/auth/google/redirect").Header("Location")
	for _, want := range []string{
		"https://accounts.google.com/",
		"client_id=client-id",
		"redirect_uri=https%3A%2F%2Fapp.trading.vukem.dev%2Fauth%2Fgoogle%2Fcallback",
		"code_challenge_method=S256",
		"state=",
	} {
		if !strings.Contains(loc, want) {
			t.Fatalf("redirect %q lacks %q", loc, want)
		}
	}
}

func TestPrivateResponsesAreNeverCached(t *testing.T) {
	t.Parallel()
	a := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, a)
	for _, path := range []string{"/", "/api/snapshot", "/api/preferences"} {
		if got := c.Get(path).AssertOK().Header("Cache-Control"); got != "private, no-store" {
			t.Fatalf("%s Cache-Control = %q", path, got)
		}
	}
}

func TestSecurity_TR001_TicketRejectsAStaleQuoteOnTheBrokerAccount(t *testing.T) {
	app := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, app).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := trading.Store{DB: db}
	old := time.Now().UTC().Add(-72 * time.Hour).Truncate(5 * time.Minute)
	candle := func(i int) trading.Candle {
		p := trading.D("190")
		return trading.Candle{
			Time: old.Add(
				time.Duration(i) * 5 * time.Minute,
			),
			Open:   p,
			High:   p,
			Low:    p,
			Close:  p,
			Volume: trading.D("1000"),
		}
	}
	ds := trading.Dataset{
		ID: "stale-aapl", Symbol: "AAPL", Currency: "USD", Source: "fixture", License: "internal test",
		Candles: []trading.Candle{candle(0), candle(1), candle(2)},
	}
	if err := store.Create(t.Context(), "datasets", ds.ID, "ready", "test", ds); err != nil {
		t.Fatal(err)
	}
	if err := store.Mutate(t.Context(), "us", "test", "arm", func(a *trading.State) error {
		a.Halted, a.Reason, a.Cash = false, "", trading.D("10000")
		a.Initial, a.DayStart, a.Peak = a.Cash, a.Cash, a.Cash
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	c.PostJSON("/api/accounts/us/ticket", map[string]any{"key": "b", "symbol": "AAPL"}).AssertUnprocessable()
	var state trading.State
	if err := store.Get(t.Context(), "trading_accounts", "us", &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Orders) != 0 {
		t.Fatalf("a stale quote must not reserve an order, got %d", len(state.Orders))
	}
}

func TestEmergencyControls(t *testing.T) {
	a := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, a).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	db, err := a.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := trading.Store{DB: db}
	load := func() trading.State {
		t.Helper()
		var state trading.State
		if err := store.Get(t.Context(), "trading_accounts", "b3", &state); err != nil {
			t.Fatal(err)
		}
		return state
	}

	// New accounts start halted, so a persisted halt shows up as a changed reason or generation.
	initial := load()
	if !initial.Halted || initial.Generation != 0 {
		t.Fatalf("unexpected initial state: %+v", initial)
	}

	c.PostJSON("/api/accounts/b3/control", map[string]any{"action": "flatten"}).AssertUnprocessable()
	if state := load(); state.Reason != initial.Reason || state.Generation != 0 {
		t.Fatalf("flatten without confirmation must not persist a halt: %+v", state)
	}

	// A refused flatten rolls the whole mutation back, including its own halt.
	c.PostJSON("/api/accounts/b3/control", map[string]any{"action": "flatten", "confirm": "ZERAR"}).
		AssertUnprocessable()
	if state := load(); state.Reason != initial.Reason || state.Generation != 0 {
		t.Fatalf("refused flatten must not persist a halt: %+v", state)
	}

	c.PostJSON("/api/accounts/b3/control", map[string]any{"action": "cancel"}).AssertOK()
	state := load()
	if !state.Halted || state.Reason != "Cancellation requested" || state.Generation != 1 {
		t.Fatalf("cancel not persisted: %+v", state)
	}

	c.PostJSON("/api/accounts/b3/control", map[string]any{"action": "nope"}).AssertUnprocessable()
}

func seedAccount(t *testing.T, store trading.Store, id string, fn func(*trading.State)) {
	t.Helper()
	err := store.Mutate(t.Context(), id, "test", "seed", func(a *trading.State) error {
		fn(a)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmergencyControlsWithOrders(t *testing.T) {
	openOrders := func(a *trading.State) {
		a.Orders = []trading.Order{
			{ID: "o1", Key: "k1", Symbol: "AAPL", Side: "buy", Status: "approved", Reserved: trading.D("100")},
			{ID: "o2", Key: "k2", Symbol: "MSFT", Side: "buy", Status: "acknowledged", Reserved: trading.D("50")},
		}
	}
	position := func(a *trading.State) {
		a.Positions = map[string]trading.Position{
			"AAPL": {Symbol: "AAPL", Qty: trading.D("1"), Average: trading.D("190"), Mark: trading.D("190")},
		}
		a.ReconciledAt = time.Now().UTC()
	}
	setup := func(t *testing.T, account string, seed func(*trading.State)) (*vukemtest.Client, trading.Store) {
		t.Helper()
		a := terminalApp(t, "owner", "owner@example.com", true)
		c := signIn(t, a).AcceptJSON()
		c.GetJSON("/api/snapshot").AssertOK()
		db, err := a.DB()
		if err != nil {
			t.Fatal(err)
		}
		store := trading.Store{DB: db}
		seedAccount(t, store, account, seed)
		return c, store
	}
	load := func(t *testing.T, store trading.Store, account string) trading.State {
		t.Helper()
		var state trading.State
		if err := store.Get(t.Context(), "trading_accounts", account, &state); err != nil {
			t.Fatal(err)
		}
		return state
	}

	t.Run("cancel cancels approved orders and marks live ones cancel_pending", func(t *testing.T) {
		c, store := setup(t, "us", openOrders)
		c.PostJSON("/api/accounts/us/control", map[string]any{"action": "cancel"}).AssertOK()
		state := load(t, store, "us")
		if state.Orders[0].Status != "canceled" || !state.Orders[0].Reserved.IsZero() {
			t.Fatalf("approved order not canceled: %+v", state.Orders[0])
		}
		if state.Orders[1].Status != "cancel_pending" {
			t.Fatalf("live order must wait for the broker: %+v", state.Orders[1])
		}
		if !state.Halted || state.Reason != "Cancellation requested" {
			t.Fatalf("cancel must halt the account: %+v", state)
		}
	})

	t.Run("cancel on the replay account cancels every open order", func(t *testing.T) {
		c, store := setup(t, "b3", openOrders)
		c.PostJSON("/api/accounts/b3/control", map[string]any{"action": "cancel"}).AssertOK()
		for _, o := range load(t, store, "b3").Orders {
			if o.Status != "canceled" {
				t.Fatalf("replay order not canceled: %+v", o)
			}
		}
	})

	t.Run("flatten is refused while a symbol has a pending order", func(t *testing.T) {
		c, store := setup(t, "us", func(a *trading.State) {
			position(a)
			a.Orders = []trading.Order{
				{ID: "o1", Key: "k1", Symbol: "AAPL", Side: "buy", Status: "acknowledged"},
			}
		})
		before := load(t, store, "us")
		c.PostJSON("/api/accounts/us/control", map[string]any{"action": "flatten", "confirm": "ZERAR"}).
			AssertUnprocessable()
		after := load(t, store, "us")
		if after.Generation != before.Generation || after.Reason != before.Reason ||
			len(after.Orders) != 1 {
			t.Fatalf("refused flatten changed the state: before=%+v after=%+v", before, after)
		}
	})
}

func TestFlattenCharacterization(t *testing.T) {
	a := terminalApp(t, "owner", "owner@example.com", true)
	c := signIn(t, a).AcceptJSON()
	c.GetJSON("/api/snapshot").AssertOK()
	db, err := a.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := trading.Store{DB: db}
	seedAccount(t, store, "us", func(a *trading.State) {
		a.Positions = map[string]trading.Position{
			"AAPL": {Symbol: "AAPL", Qty: trading.D("1"), Average: trading.D("190"), Mark: trading.D("190")},
		}
		a.ReconciledAt = time.Now().UTC()
	})
	// Characterization: a confirmed flatten on a reconciled account halts it and reserves one sell order per position.
	c.PostJSON("/api/accounts/us/control", map[string]any{"action": "flatten", "confirm": "ZERAR"}).AssertOK()
	var state trading.State
	if err := store.Get(t.Context(), "trading_accounts", "us", &state); err != nil {
		t.Fatal(err)
	}
	if !state.Halted || state.Reason != "Flatten requested; monitor orders and positions" || len(state.Orders) != 1 {
		t.Fatalf("unexpected flatten state: %+v", state)
	}
	if o := state.Orders[0]; o.Side != "sell" || o.Symbol != "AAPL" || o.Status != "approved" ||
		!strings.HasSuffix(o.Key, "/flatten") {
		t.Fatalf("unexpected flatten order: %+v", o)
	}
}
