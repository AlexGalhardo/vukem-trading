package adapters

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

func TestMoney_AlpacaRefusesLiveAndForeignHostsBeforeAnyRequest(t *testing.T) {
	t.Parallel()
	for _, host := range []string{
		"https://api.alpaca.markets", "https://broker-api.alpaca.markets", "https://example.com",
	} {
		a := Alpaca{Key: "k", Secret: "s", PaperURL: host, DataURL: host, Client: &http.Client{
			Transport: transport(func(*http.Request) (*http.Response, error) {
				t.Fatal("network called for", host)
				return nil, nil
			}),
		}}
		if _, err := a.Account(context.Background()); err == nil {
			t.Fatalf("%s accepted as paper host", host)
		}
		if _, err := a.DailyBars(context.Background(), []string{"AAPL"}, time.Now(), time.Now()); err == nil {
			t.Fatalf("%s accepted as data host", host)
		}
	}
}

func TestAlpacaDailyBarsPagesAndKeysByExchangeDate(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/v2/stocks/bars" || q.Get("timeframe") != "1Day" || q.Get("feed") != "iex" ||
			q.Get("symbols") != "AAPL,MSFT" {
			http.Error(w, "bad query", http.StatusBadRequest)
			return
		}
		if calls.Add(1) == 1 {
			_, _ = w.Write([]byte(`{"bars":{"AAPL":[` +
				`{"t":"2026-11-11T05:00:00Z","o":"10","h":"11","l":"9","c":"10.5","v":"1000"},` +
				`{"t":"2026-11-12T05:00:00Z","o":"10.5","h":"12","l":"10","c":"11","v":"900"}]},"next_page_token":"p2"}`))
			return
		}
		if q.Get("page_token") != "p2" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"bars":{"MSFT":[` +
			`{"t":"2026-11-12T05:00:00Z","o":"20","h":"21","l":"19","c":"20.5","v":"50"}]},"next_page_token":null}`))
	}))
	t.Cleanup(srv.Close)
	a := Alpaca{Key: "k", Secret: "s", DataURL: srv.URL}
	got, err := a.DailyBars(context.Background(), []string{"AAPL", "MSFT"}, time.Now().AddDate(0, 0, -5), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(got["AAPL"]) != 2 || len(got["MSFT"]) != 1 {
		t.Fatalf("bars = %v", got)
	}
	if day := got["AAPL"][0].Time.Format(time.DateOnly); day != "2026-11-11" {
		t.Fatalf("day = %s", day)
	}
	if !got["AAPL"][1].Close.Equal(d.RequireFromString("11")) {
		t.Fatal("close")
	}
}

func TestAlpacaDailyBarsRejectsInvalidBars(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(
			[]byte(`{"bars":{"AAPL":[{"t":"2026-11-11T05:00:00Z","o":"10","h":"8","l":"9","c":"10","v":"1"}]}}`),
		)
	}))
	t.Cleanup(srv.Close)
	a := Alpaca{Key: "k", Secret: "s", DataURL: srv.URL}
	if _, err := a.DailyBars(context.Background(), []string{"AAPL"}, time.Now(), time.Now()); err == nil {
		t.Fatal("invalid candle accepted")
	}
}

func TestAlpacaStopProtectedEntrySendsAnOTOOrder(t *testing.T) {
	t.Parallel()
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"id":"1","client_order_id":"c1","status":"accepted"}`))
	}))
	t.Cleanup(srv.Close)
	a := Alpaca{Key: "k", Secret: "s", PaperURL: srv.URL}
	o := trading.Order{ID: "c1", Symbol: "AAPL", Side: "buy", Qty: trading.D("3"), Limit: trading.D("100")}
	if _, err := a.SubmitWithStop(context.Background(), o, trading.D("90")); err != nil {
		t.Fatal(err)
	}
	stop, _ := body["stop_loss"].(map[string]any)
	if body["order_class"] != "oto" || stop["stop_price"] != "90.00" || body["client_order_id"] != "c1" {
		t.Fatalf("payload = %v", body)
	}
	fractional := trading.Order{ID: "c", Symbol: "AAPL", Side: "buy", Qty: trading.D("1.5"), Limit: trading.D("100")}
	sell := trading.Order{ID: "c", Symbol: "AAPL", Side: "sell", Qty: trading.D("1"), Limit: trading.D("100")}
	for name, bad := range map[string]struct {
		o    trading.Order
		stop d.Decimal
	}{
		"stop above limit": {o, trading.D("110")},
		"fractional":       {fractional, trading.D("90")},
		"sell":             {sell, trading.D("90")},
	} {
		if _, err := a.SubmitWithStop(context.Background(), bad.o, bad.stop); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestTestnetStopLeg(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	var seen url.Values
	srv := signedServer(
		t, &requests, &seen,
		`{"orderId":9,"clientOrderId":"s1","status":"NEW","executedQty":"0","cummulativeQuoteQty":"0"}`,
	)
	t.Cleanup(srv.Close)
	b := testnet(srv.URL)
	qty, stop := d.RequireFromString("0.01"), d.RequireFromString("90")
	if _, err := b.SubmitStop(context.Background(), "BTCUSDT", qty, stop, d.RequireFromString("89"), "s1"); err != nil {
		t.Fatal(err)
	}
	if seen.Get("type") != "STOP_LOSS_LIMIT" || seen.Get("side") != "SELL" || seen.Get("stopPrice") != "90" {
		t.Fatalf("query = %v", seen)
	}
	if _, err := b.SubmitStop(context.Background(), "BTCUSDT", qty, stop, d.RequireFromString("91"), "s1"); err == nil {
		t.Fatal("limit above stop accepted")
	}
}

func TestTestnetFiltersAreReadFromExchangeInfo(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/exchangeInfo" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"symbols":[{"symbol":"BTCUSDT","filters":[` +
			`{"filterType":"PRICE_FILTER","tickSize":"0.01000000"},` +
			`{"filterType":"LOT_SIZE","minQty":"0.00001000","stepSize":"0.00001000"},` +
			`{"filterType":"NOTIONAL","minNotional":"5.00000000"}]}]}`))
	}))
	t.Cleanup(srv.Close)
	f, err := testnet(srv.URL).Filters(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if !f.Tick.Equal(d.RequireFromString("0.01")) || !f.Step.Equal(d.RequireFromString("0.00001")) ||
		!f.MinNotional.Equal(d.RequireFromString("5")) {
		t.Fatalf("filters = %+v", f)
	}
	if _, err := testnet(srv.URL).Filters(context.Background(), "ETHUSDT"); err == nil {
		t.Fatal("unknown symbol accepted")
	}
}

func TestSendMessageUsesTheBotEndpointAndHidesTheToken(t *testing.T) {
	t.Parallel()
	var path, text string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		var in map[string]string
		_ = json.NewDecoder(r.Body).Decode(&in)
		text = in["text"]
		http.Error(w, "secret-body", http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	err := SendMessage(context.Background(), srv.URL, "123:abc", "42", "hello")
	if path != "/bot123:abc/sendMessage" || text != "hello" {
		t.Fatalf("path=%q text=%q", path, text)
	}
	if err == nil || strings.Contains(err.Error(), "123:abc") || strings.Contains(err.Error(), "secret-body") {
		t.Fatalf("err = %v", err)
	}
	if err := SendMessage(context.Background(), "https://example.com", "123:abc", "42", "x"); err == nil {
		t.Fatal("foreign API host accepted")
	}
	if err := SendMessage(context.Background(), srv.URL, "", "42", "x"); err != nil {
		t.Fatal("no token must be a no-op")
	}
}
