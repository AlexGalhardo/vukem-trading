package paper

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/adapters"
	"github.com/AlexGalhardo/vukem-trading/internal/reco"
)

type fakeBroker struct {
	acc     Account
	accErr  error
	failFor string
	entries []Entry
}

func (*fakeBroker) Name() string { return "fake" }

func (f *fakeBroker) Account(context.Context) (Account, error) { return f.acc, f.accErr }

func (f *fakeBroker) Enter(_ context.Context, e Entry) (Placed, error) {
	if e.Symbol == f.failFor {
		return Placed{}, errors.New("rejected")
	}
	f.entries = append(f.entries, e)
	return Placed{Qty: d.NewFromInt(1)}, nil
}

func top(symbols ...string) []reco.Entry {
	out := make([]reco.Entry, 0, len(symbols))
	for i, s := range symbols {
		out = append(out, reco.Entry{Rank: i + 1, Symbol: s, LastClose: d.NewFromInt(100)})
	}
	return out
}

func TestRunSizesSlotsSkipsHeldAndContinuesAfterAFailure(t *testing.T) {
	t.Parallel()
	b := &fakeBroker{
		acc:     Account{Equity: d.NewFromInt(100000), Held: map[string]bool{"B": true}},
		failFor: "C",
	}
	cfg := reco.USStrategy()
	res, err := Run(t.Context(), b, "2026-11-14", top("A", "B", "C", "D", "E", "F", "G"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != cfg.HoldTop {
		t.Fatalf("results = %d, want HoldTop %d", len(res), cfg.HoldTop)
	}
	if res[1].Skip == "" || res[2].Err == nil || res[0].Placed == nil || res[3].Placed == nil {
		t.Fatalf("results = %+v", res)
	}
	for _, e := range b.entries {
		if !e.Budget.Equal(d.NewFromInt(20000)) || e.StopBPS != cfg.StopBPS {
			t.Fatalf("entry = %+v", e)
		}
		if e.ClientID != "reco-2026-11-14-"+e.Symbol {
			t.Fatalf("client id = %q", e.ClientID)
		}
	}
}

func TestRunRefusesAnEmptyRankingAndAnAccountWithoutEquity(t *testing.T) {
	t.Parallel()
	cfg := reco.USStrategy()
	if _, err := Run(t.Context(), &fakeBroker{}, "x", nil, cfg); err == nil {
		t.Fatal("empty ranking accepted")
	}
	if _, err := Run(t.Context(), &fakeBroker{}, "x", top("A"), cfg); err == nil {
		t.Fatal("zero equity accepted")
	}
	if _, err := Run(t.Context(), &fakeBroker{accErr: errors.New("down")}, "x", top("A"), cfg); err == nil {
		t.Fatal("account error ignored")
	}
}

func TestEnabledNeedsTheExplicitSwitch(t *testing.T) {
	t.Parallel()
	for v, want := range map[string]bool{"": false, "false": false, "1": false, "true": true, "TRUE": true} {
		if got := Enabled(func(string) string { return v }); got != want {
			t.Fatalf("PAPER_TRADING=%q: %v", v, got)
		}
	}
}

func TestMoney_NoLiveEndpointCanBeUsedByThePaperBrokers(t *testing.T) {
	t.Parallel()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	t.Cleanup(srv.Close)
	live := []string{
		"https://api.alpaca.markets", "https://broker-api.alpaca.markets",
		"https://api.binance.com", "https://api1.binance.com", "https://fapi.binance.com", "https://example.com",
	}
	for _, host := range live {
		alpaca := AlpacaPaper{A: adapters.Alpaca{Key: "k", Secret: "s", PaperURL: host, DataURL: host}}
		if _, err := alpaca.Account(t.Context()); err == nil {
			t.Fatalf("Alpaca accepted %s", host)
		}
		_, err := alpaca.Enter(t.Context(), Entry{
			Symbol: "AAPL", LastClose: d.NewFromInt(100), Budget: d.NewFromInt(1000), StopBPS: 500, ClientID: "x",
		})
		if err == nil {
			t.Fatalf("Alpaca placed an order on %s", host)
		}
		bin := BinancePaper{T: adapters.BinanceTestnet{Key: "k", Secret: "s", BaseURL: host}}
		if _, err := bin.Account(t.Context()); err == nil {
			t.Fatalf("Binance accepted %s", host)
		}
		_, err = bin.Enter(t.Context(), Entry{
			Symbol: "BTCUSDT", LastClose: d.NewFromInt(100), Budget: d.NewFromInt(1000), StopBPS: 500, ClientID: "x",
		})
		if err == nil {
			t.Fatalf("Binance placed an order on %s", host)
		}
	}
	if hits != 0 {
		t.Fatal("a request reached the fake server")
	}
}

func TestAlpacaPaperEntersWholeSharesWithAnAttachedStop(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var posted map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/account":
			_, _ = w.Write([]byte(`{"id":"1","equity":"100000","cash":"100000","trading_blocked":false}`))
		case r.URL.Path == "/v2/positions":
			_, _ = w.Write([]byte(`[{"symbol":"MSFT","qty":"3","avg_entry_price":"1","current_price":"1"}]`))
		case r.URL.Path == "/v2/orders" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":"o","symbol":"NVDA","status":"new"}]`))
		case r.URL.Path == "/v2/orders" && r.Method == http.MethodPost:
			raw, _ := io.ReadAll(r.Body)
			mu.Lock()
			_ = json.Unmarshal(raw, &posted)
			mu.Unlock()
			_, _ = w.Write([]byte(`{"id":"order-1","client_order_id":"x","status":"accepted"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	b := AlpacaPaper{A: adapters.Alpaca{Key: "k", Secret: "s", PaperURL: srv.URL}}
	res, err := Run(t.Context(), b, "2026-11-14", []reco.Entry{
		{Rank: 1, Symbol: "MSFT", LastClose: d.NewFromInt(400)},
		{Rank: 2, Symbol: "NVDA", LastClose: d.NewFromInt(200)},
		{Rank: 3, Symbol: "AAPL", LastClose: d.RequireFromString("187.37")},
	}, reco.USStrategy())
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Skip == "" || res[1].Skip == "" || res[2].Placed == nil {
		t.Fatalf("results = %+v", res)
	}
	stop, _ := posted["stop_loss"].(map[string]any)
	if posted["order_class"] != "oto" || posted["symbol"] != "AAPL" || stop["stop_price"] != "168.64" ||
		posted["client_order_id"] != "reco-2026-11-14-AAPL" || posted["qty"] != "106" {
		t.Fatalf("order = %v", posted)
	}
}

// testnetServer fakes the three endpoints the crypto entry uses; fill controls the entry status.
func testnetServer(t *testing.T, status string, rejectStop bool, calls *[]string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		q := r.URL.Query()
		*calls = append(*calls, r.Method+" "+r.URL.Path+" "+q.Get("type"))
		switch {
		case r.URL.Path == "/api/v3/exchangeInfo":
			_, _ = w.Write([]byte(`{"symbols":[{"symbol":"BTCUSDT","filters":[` +
				`{"filterType":"PRICE_FILTER","tickSize":"0.01"},` +
				`{"filterType":"LOT_SIZE","minQty":"0.00001","stepSize":"0.00001"},` +
				`{"filterType":"NOTIONAL","minNotional":"5"}]}]}`))
		case r.URL.Path == "/api/v3/account":
			_, _ = w.Write(
				[]byte(`{"canTrade":true,"balances":[{"asset":"USDT","free":"9000"},{"asset":"ETH","free":"1"}]}`),
			)
		case r.URL.Path == "/api/v3/order" && r.Method == http.MethodPost && q.Get("type") == "LIMIT":
			filled := "0"
			if status == "FILLED" {
				filled = q.Get("quantity")
			}
			_, _ = w.Write(
				[]byte(`{"orderId":1,"clientOrderId":"` + q.Get("newClientOrderId") + `","status":"` + status +
					`","executedQty":"` + filled + `","cummulativeQuoteQty":"0"}`),
			)
		case r.URL.Path == "/api/v3/order" && r.Method == http.MethodPost:
			if rejectStop {
				http.Error(w, "no", http.StatusBadRequest)
				return
			}
			_, _ = w.Write(
				[]byte(`{"orderId":2,"clientOrderId":"s","status":"NEW","executedQty":"0","cummulativeQuoteQty":"0"}`),
			)
		case r.URL.Path == "/api/v3/order" && r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{"orderId":1,"origClientOrderId":"` + q.Get("origClientOrderId") +
				`","status":"CANCELED","executedQty":"0","cummulativeQuoteQty":"0"}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func binance(base string) BinancePaper {
	fixed := time.Date(2026, 11, 14, 0, 30, 0, 0, time.UTC)
	return BinancePaper{
		T: adapters.BinanceTestnet{Key: "k", Secret: "s", BaseURL: base, Now: func() time.Time { return fixed }},
	}
}

var btcEntry = Entry{
	Symbol: "BTCUSDT", LastClose: d.NewFromInt(70000), Budget: d.NewFromInt(3000), StopBPS: 1500, SlippageBPS: 5,
	ClientID: "reco-2026-11-14-BTCUSDT",
}

func TestBinancePaperFilledEntryGetsItsStopLeg(t *testing.T) {
	t.Parallel()
	var calls []string
	srv := testnetServer(t, "FILLED", false, &calls)
	t.Cleanup(srv.Close)
	b := binance(srv.URL)
	acc, err := b.Account(t.Context())
	if err != nil || !acc.Equity.Equal(d.NewFromInt(9000)) || !acc.Held["ETHUSDT"] {
		t.Fatalf("account = %+v err=%v", acc, err)
	}
	p, err := b.Enter(t.Context(), btcEntry)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Stop.Equal(d.NewFromInt(59500)) || !p.Qty.IsPositive() {
		t.Fatalf("placed = %+v", p)
	}
	if got := strings.Join(
		calls[len(calls)-2:],
		"|",
	); got != "POST /api/v3/order LIMIT|POST /api/v3/order STOP_LOSS_LIMIT" {
		t.Fatalf("calls = %v", calls)
	}
}

func TestBinancePaperCancelsAnUnfilledEntryAndNeverLeavesItUnprotected(t *testing.T) {
	t.Parallel()
	var calls []string
	srv := testnetServer(t, "NEW", false, &calls)
	t.Cleanup(srv.Close)
	if _, err := binance(srv.URL).Enter(t.Context(), btcEntry); err == nil {
		t.Fatal("unfilled entry reported as placed")
	}
	joined := strings.Join(calls, "|")
	if !strings.Contains(joined, "DELETE /api/v3/order") || strings.Contains(joined, "STOP_LOSS_LIMIT") {
		t.Fatalf("calls = %v", calls)
	}
}

func TestBinancePaperReportsAFilledEntryWhoseStopWasRejected(t *testing.T) {
	t.Parallel()
	var calls []string
	srv := testnetServer(t, "FILLED", true, &calls)
	t.Cleanup(srv.Close)
	_, err := binance(srv.URL).Enter(t.Context(), btcEntry)
	if err == nil || !strings.Contains(err.Error(), "UNPROTECTED") {
		t.Fatalf("err = %v", err)
	}
}

func TestBinancePaperRejectsABudgetBelowTheMinimum(t *testing.T) {
	t.Parallel()
	var calls []string
	srv := testnetServer(t, "FILLED", false, &calls)
	t.Cleanup(srv.Close)
	e := btcEntry
	e.Budget = d.NewFromInt(1)
	if _, err := binance(srv.URL).Enter(t.Context(), e); err == nil {
		t.Fatal("dust order accepted")
	}
	for _, c := range calls {
		if strings.HasPrefix(c, "POST") {
			t.Fatalf("an order was sent: %v", calls)
		}
	}
}
