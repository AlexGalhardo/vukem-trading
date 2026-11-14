// Package adapters isolates broker/network protocols from the trading domain.
package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

const (
	alpacaPaperHost = "paper-api.alpaca.markets"
	alpacaDataHost  = "data.alpaca.markets"
)

// Alpaca talks to the Alpaca paper-trading and market-data APIs. It refuses every other host, so the
// live trading endpoint cannot be configured by mistake: PaperURL and DataURL exist for tests and only
// accept loopback addresses.
type Alpaca struct {
	Key, Secret string
	Client      *http.Client
	PaperURL    string
	DataURL     string
}

func (a Alpaca) base(data bool) (string, error) {
	raw, allowed := a.PaperURL, alpacaPaperHost
	if data {
		raw, allowed = a.DataURL, alpacaDataHost
	}
	if raw == "" {
		return "https://" + allowed, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", errors.New("invalid Alpaca address")
	}
	if h := u.Hostname(); h != allowed && !isLoopback(h) {
		return "", errors.New("only the Alpaca paper and market-data hosts are allowed")
	}
	return u.Scheme + "://" + u.Host, nil
}

func isLoopback(host string) bool {
	ip := net.ParseIP(host)
	return host == "localhost" || (ip != nil && ip.IsLoopback())
}

func (a Alpaca) request(ctx context.Context, method, path string, input, output any, data bool) error {
	if a.Key == "" || a.Secret == "" {
		return errors.New("Alpaca Paper credentials missing")
	}
	host, err := a.base(data)
	if err != nil {
		return err
	}
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, host+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("APCA-API-KEY-ID", a.Key)
	req.Header.Set("APCA-API-SECRET-KEY", a.Secret)
	req.Header.Set("Content-Type", "application/json")
	client := a.Client
	if client == nil {
		client = &http.Client{
			Timeout:       15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("broker without confirmation: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Alpaca HTTP %d", resp.StatusCode)
	}
	if output != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(output)
	}
	return nil
}

type BrokerAccount struct {
	ID             string    `json:"id"`
	Cash           d.Decimal `json:"cash"`
	Equity         d.Decimal `json:"equity"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	TradingBlocked bool      `json:"trading_blocked"`
}
type BrokerOrder struct {
	ID       string     `json:"id"`
	ClientID string     `json:"client_order_id"`
	Symbol   string     `json:"symbol"`
	Side     string     `json:"side"`
	Qty      d.Decimal  `json:"qty"`
	Filled   d.Decimal  `json:"filled_qty"`
	Average  *d.Decimal `json:"filled_avg_price"`
	Status   string     `json:"status"`
	Updated  time.Time  `json:"updated_at"`
}
type BrokerPosition struct {
	Symbol  string    `json:"symbol"`
	Qty     d.Decimal `json:"qty"`
	Average d.Decimal `json:"avg_entry_price"`
	Price   d.Decimal `json:"current_price"`
}
type Clock struct {
	Open      bool      `json:"is_open"`
	NextClose time.Time `json:"next_close"`
	Time      time.Time `json:"timestamp"`
}

func (a Alpaca) Account(ctx context.Context) (BrokerAccount, error) {
	var out BrokerAccount
	err := a.request(ctx, "GET", "/v2/account", nil, &out, false)
	return out, err
}

func (a Alpaca) Clock(ctx context.Context) (Clock, error) {
	var out Clock
	err := a.request(ctx, "GET", "/v2/clock", nil, &out, false)
	return out, err
}

func (a Alpaca) Positions(ctx context.Context) ([]BrokerPosition, error) {
	out := []BrokerPosition{}
	err := a.request(ctx, "GET", "/v2/positions", nil, &out, false)
	return out, err
}

func (a Alpaca) Orders(ctx context.Context) ([]BrokerOrder, error) {
	out := []BrokerOrder{}
	err := a.request(ctx, "GET", "/v2/orders?status=open&limit=500", nil, &out, false)
	return out, err
}

func (a Alpaca) Lookup(ctx context.Context, id string) (BrokerOrder, error) {
	var out BrokerOrder
	err := a.request(ctx, "GET", "/v2/orders:by_client_order_id?client_order_id="+url.QueryEscape(id), nil, &out, false)
	return out, err
}

func (a Alpaca) Submit(ctx context.Context, o trading.Order) (BrokerOrder, error) {
	var out BrokerOrder
	payload := map[string]any{
		"symbol":          o.Symbol,
		"qty":             o.Qty.String(),
		"side":            o.Side,
		"type":            "limit",
		"limit_price":     o.Limit.StringFixed(2),
		"time_in_force":   "day",
		"client_order_id": o.ID,
	}
	err := a.request(ctx, "POST", "/v2/orders", payload, &out, false)
	return out, err
}

func (a Alpaca) Cancel(ctx context.Context, id string) error {
	return a.request(ctx, "DELETE", "/v2/orders/"+url.PathEscape(id), nil, nil, false)
}

func (a Alpaca) Bars(ctx context.Context, symbol string, start, end time.Time) (trading.Dataset, error) {
	i, err := trading.FindInstrument(symbol)
	if err != nil || i.Market != "US" {
		return trading.Dataset{}, errors.New("invalid symbol for Alpaca")
	}
	params := url.Values{
		"timeframe":  {"5Min"},
		"start":      {start.UTC().Format(time.RFC3339)},
		"end":        {end.UTC().Format(time.RFC3339)},
		"feed":       {"iex"},
		"adjustment": {"raw"},
		"limit":      {"10000"},
	}
	calendar, err := a.Calendar(ctx, start, end)
	if err != nil {
		return trading.Dataset{}, err
	}
	ds := trading.Dataset{
		CalendarSource: "Alpaca exchange calendar",
		SessionCloses:  calendar,
		ID:             trading.ID(),
		Symbol:         symbol,
		Currency:       "USD",
		Timeframe:      5,
		Source:         "Alpaca IEX",
		License:        "Alpaca account use; confirm contracted terms",
		CreatedAt:      time.Now().UTC(),
		Candles:        []trading.Candle{},
	}
	for page := range 20 {
		var response struct {
			Bars []struct {
				Time   time.Time `json:"t"`
				Open   d.Decimal `json:"o"`
				High   d.Decimal `json:"h"`
				Low    d.Decimal `json:"l"`
				Close  d.Decimal `json:"c"`
				Volume d.Decimal `json:"v"`
			} `json:"bars"`
			Next string `json:"next_page_token"`
		}
		if err := a.request(
			ctx,
			"GET",
			"/v2/stocks/"+symbol+"/bars?"+params.Encode(),
			nil,
			&response,
			true,
		); err != nil {
			return ds, err
		}
		for _, b := range response.Bars {
			c := trading.Candle{Time: b.Time, Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume}
			if err := c.Validate(); err != nil {
				return ds, err
			}
			sessionClose, exists := calendar[trading.MarketDay("US", c.Time)]
			if exists && trading.Session("US", c.Time) && !c.Time.Add(5*time.Minute).After(end) &&
				!c.Time.Add(5*time.Minute).After(sessionClose) {
				if len(ds.Candles) > 0 && !c.Time.After(ds.Candles[len(ds.Candles)-1].Time) {
					return ds, errors.New("candles duplicated/out of order")
				}
				ds.Candles = append(ds.Candles, c)
			}
		}
		if response.Next == "" {
			break
		}
		if len(ds.Candles) > 100000 || page == 19 {
			return ds, errors.New("history exceeds limit; select a shorter period")
		}
		params.Set("page_token", response.Next)
	}
	if len(ds.Candles) < 3 {
		return ds, errors.New("insufficient IEX history for the period")
	}
	ds.Hash = trading.DatasetHash(ds.Candles)
	return ds, nil
}

func (a Alpaca) Calendar(ctx context.Context, start, end time.Time) (map[string]time.Time, error) {
	var rows []struct {
		Date  string `json:"date"`
		Open  string `json:"open"`
		Close string `json:"close"`
	}
	params := url.Values{"start": {start.Format("2006-01-02")}, "end": {end.Format("2006-01-02")}}
	if err := a.request(ctx, "GET", "/v2/calendar?"+params.Encode(), nil, &rows, false); err != nil {
		return nil, err
	}
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, err
	}
	out := map[string]time.Time{}
	for _, row := range rows {
		closeAt, err := time.ParseInLocation("2006-01-02 15:04", row.Date+" "+row.Close, zone)
		if err != nil {
			return nil, errors.New("invalid calendar")
		}
		out[row.Date] = closeAt.UTC()
	}
	return out, nil
}
