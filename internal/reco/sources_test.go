package reco

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBinanceDailyKeepsOnlyClosedDaysAndSkipsUnknownPairs(t *testing.T) {
	t.Parallel()
	now := time.UnixMilli(1_794_700_800_000 + 3600_000).UTC() // one hour into the UTC day that starts at 1794700800000
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("interval") != "1d" {
			http.Error(w, "interval", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("symbol") == "GONEUSDT" {
			http.Error(w, `{"code":-1121}`, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`[` +
			`[1794528000000,"71460.0","71514.0","67568.0","68314.0","29648.7",1794614399999,"0",1,"0","0","0"],` +
			`[1794614400000,"68314.0","68446.0","65000.0","66139.4","21826.7",1794700799999,"0",1,"0","0","0"],` +
			`[1794700800000,"66139.4","66500.0","65000.0","66000.0","10.0",1794787199999,"0",1,"0","0","0"]]`))
	}))
	t.Cleanup(srv.Close)
	got, err := BinanceDaily{BaseURL: srv.URL, Now: func() time.Time { return now }}.
		Daily(context.Background(), []string{"BTCUSDT", "GONEUSDT"}, now.AddDate(0, 0, -5), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["GONEUSDT"]; ok || len(got["BTCUSDT"]) != 2 {
		t.Fatalf("got %v", got)
	}
	if got["BTCUSDT"][1].Time.Format(time.DateOnly) != "2026-11-14" {
		t.Fatalf("day = %v", got["BTCUSDT"][1].Time)
	}
}

func TestBinanceDailyFailsOnServerErrors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	_, err := BinanceDaily{BaseURL: srv.URL}.Daily(context.Background(), []string{"BTCUSDT"}, time.Now(), time.Now())
	if err == nil {
		t.Fatal("server error ignored")
	}
}

func TestAlpacaDailyNeedsCredentialsAndDropsAnOpenSession(t *testing.T) {
	t.Parallel()
	if _, err := (AlpacaDaily{}).Daily(context.Background(), []string{"AAPL"}, time.Now(), time.Now()); err == nil {
		t.Fatal("missing credentials accepted")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"bars":{"AAPL":[` +
			`{"t":"2026-11-12T05:00:00Z","o":"10","h":"11","l":"9","c":"10.5","v":"1000"},` +
			`{"t":"2026-11-13T05:00:00Z","o":"10.5","h":"12","l":"10","c":"11","v":"900"}]}}`))
	}))
	t.Cleanup(srv.Close)
	ny, _ := time.LoadLocation("America/New_York")
	for _, tc := range []struct {
		name string
		now  time.Time
		want int
	}{
		{"mid-session", time.Date(2026, 11, 13, 11, 0, 0, 0, ny), 1},
		{"after the close", time.Date(2026, 11, 13, 19, 30, 0, 0, ny), 2},
	} {
		a := AlpacaDaily{Key: "k", Secret: "s", DataURL: srv.URL, Now: func() time.Time { return tc.now }}
		got, err := a.Daily(context.Background(), []string{"AAPL"}, tc.now.AddDate(0, 0, -5), tc.now)
		if err != nil {
			t.Fatal(err)
		}
		if len(got["AAPL"]) != tc.want {
			t.Fatalf("%s: %d bars, want %d", tc.name, len(got["AAPL"]), tc.want)
		}
	}
}

func TestEveryMessageStartsAndEndsWithTheDisclaimer(t *testing.T) {
	t.Parallel()
	rep := Report{Date: "2026-11-13", Disclaimer: Disclaimer, Markets: []MarketReport{
		{Market: MarketUS, Source: "x", Ranking: Rank(universe(), USRank(), day0.AddDate(0, 0, 119))},
		{Market: MarketCrypto, Source: "y", Error: "boom"},
		{Market: "OTHER", Source: "z"},
	}}
	msgs := Messages(rep)
	if len(msgs) != 3 {
		t.Fatalf("messages = %d", len(msgs))
	}
	for i, m := range msgs {
		if !strings.HasPrefix(m, Disclaimer) || !strings.HasSuffix(m, Disclaimer) {
			t.Fatalf("message %d lacks the disclaimer: %q", i, m)
		}
	}
}
