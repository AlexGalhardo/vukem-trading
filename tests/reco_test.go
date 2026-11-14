package tests

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"
	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/app/commands"
	"github.com/AlexGalhardo/vukem-trading/bootstrap"
	"github.com/AlexGalhardo/vukem-trading/internal/reco"
	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

type fakeSource struct {
	name string
	err  error
}

func (f fakeSource) Name() string { return f.name }

func (f fakeSource) Daily(_ context.Context, symbols []string, _, to time.Time) (map[string][]trading.Candle, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[string][]trading.Candle{}
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	for k, s := range symbols {
		price := 100.0
		for i := range 90 {
			open := price
			price *= 1 + 0.001*float64(k%7) + 0.01*float64((i*(k+3))%5-2)/5
			out[s] = append(out[s], trading.Candle{
				Time: end.AddDate(0, 0, i-89),
				Open: d.NewFromFloat(open).Round(4), High: d.NewFromFloat(max(open, price) * 1.01).Round(4),
				Low: d.NewFromFloat(min(open, price) * 0.99).Round(4), Close: d.NewFromFloat(price).Round(4),
				Volume: d.NewFromInt(5_000_000),
			})
		}
	}
	return out, nil
}

func TestRecoDailyStoresSendsAndServesTheReport(t *testing.T) {
	a := terminalApp(t, "owner", "owner@example.com", true)
	db, err := a.DB()
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var sent []string
	tg := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var in map[string]string
		_ = json.Unmarshal(raw, &in)
		mu.Lock()
		sent = append(sent, r.URL.Path+"|"+in["chat_id"]+"|"+in["text"])
		mu.Unlock()
	}))
	t.Cleanup(tg.Close)
	env := map[string]string{"TELEGRAM_BOT_TOKEN": "1:tok", "TELEGRAM_CHAT_ID": "42"}
	now := time.Date(2026, 11, 14, 0, 30, 0, 0, time.UTC)
	deps := reco.DailyDeps{
		Env: func(k string) string { return env[k] }, Store: reco.Store{DB: db}, Log: slog.New(slog.DiscardHandler),
		Now: func() time.Time { return now }, TelegramBase: tg.URL,
		Sources: &reco.Sources{US: fakeSource{name: "fake us"}, Crypto: fakeSource{name: "fake crypto"}},
	}
	if err := reco.Daily(t.Context(), deps); err != nil {
		t.Fatal(err)
	}
	if err := reco.Daily(t.Context(), deps); err != nil {
		t.Fatal("a rerun on the same date must replace it:", err)
	}
	if len(sent) != 4 {
		t.Fatalf("messages = %d, want 2 per run", len(sent))
	}
	for _, m := range sent {
		if !strings.HasPrefix(m, "/bot1:tok/sendMessage|42|"+reco.Disclaimer) {
			t.Fatalf("message without the disclaimer: %q", m)
		}
	}
	rep, ok, err := reco.Store{DB: db}.Latest(t.Context())
	if err != nil || !ok || rep.Date != "2026-11-14" {
		t.Fatalf("latest = %+v ok=%v err=%v", rep, ok, err)
	}
	if len(rep.Markets) != 2 || len(rep.Markets[0].Ranking.Top) != 20 || len(rep.Markets[1].Ranking.Top) != 10 {
		t.Fatalf("ranking sizes: %d/%d", len(rep.Markets[0].Ranking.Top), len(rep.Markets[1].Ranking.Top))
	}
	if rep.Disclaimer != reco.Disclaimer {
		t.Fatal("stored report lost the disclaimer")
	}

	c := signIn(t, a).AcceptJSON()
	c.GetJSON("/api/recommendations/latest").AssertOK().AssertJSONPath("date", "2026-11-14").
		AssertJSONPath("disclaimer", reco.Disclaimer)
	signIn(t, a).Get("/recommendations").AssertOK().AssertInertia("Terminal")
}

func TestRecoDailyWithoutTelegramAndWithOneMarketDown(t *testing.T) {
	a := terminalApp(t, "owner", "owner@example.com", true)
	db, err := a.DB()
	if err != nil {
		t.Fatal(err)
	}
	deps := reco.DailyDeps{
		Env: func(string) string { return "" }, Store: reco.Store{DB: db}, Log: slog.New(slog.DiscardHandler),
		Now: func() time.Time { return time.Date(2026, 11, 14, 0, 30, 0, 0, time.UTC) },
		Sources: &reco.Sources{
			US: fakeSource{name: "us", err: errors.New("no key")}, Crypto: fakeSource{name: "crypto"},
		},
	}
	if err := reco.Daily(t.Context(), deps); err != nil {
		t.Fatal(err)
	}
	rep, _, _ := reco.Store{DB: db}.Latest(t.Context())
	if rep.Markets[0].Error == "" || len(rep.Markets[1].Ranking.Top) != 10 {
		t.Fatalf("one market down must not stop the other: %+v", rep.Markets)
	}
	deps.Sources.Crypto = fakeSource{name: "crypto", err: errors.New("down")}
	if err := reco.Daily(t.Context(), deps); err == nil {
		t.Fatal("all markets down must fail the job")
	}
}

func TestRecoEndpointIsPrivateAndCarriesTheDisclaimerWhenEmpty(t *testing.T) {
	a := terminalApp(t, "owner", "owner@example.com", true)
	a.Client(t).AcceptJSON().GetJSON("/api/recommendations/latest").AssertUnauthorized()
	signIn(t, a).AcceptJSON().GetJSON("/api/recommendations/latest").AssertOK().
		AssertJSONPath("disclaimer", reco.Disclaimer)
}

func TestRecoPaperIsOffByDefaultAndNeedsAFreshRanking(t *testing.T) {
	a := terminalApp(t, "owner", "owner@example.com", true)
	if err := (commands.RecoPaper{}).Run(t.Context(), a.App, nil); err != nil {
		t.Fatal("paper placement must be a no-op without PAPER_TRADING:", err)
	}
	on := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"PAPER_TRADING": "true"}})
	if err := (commands.RecoPaper{}).Run(t.Context(), on.App, nil); err == nil {
		t.Fatal("paper placement without a stored ranking must fail")
	}
	db, err := on.DB()
	if err != nil {
		t.Fatal(err)
	}
	old := reco.Report{Date: "2020-01-02", Disclaimer: reco.Disclaimer}
	if err := (reco.Store{DB: db}).Save(t.Context(), old); err != nil {
		t.Fatal(err)
	}
	if err := (commands.RecoPaper{}).Run(t.Context(), on.App, nil); err == nil {
		t.Fatal("a stale ranking was traded")
	}
}
