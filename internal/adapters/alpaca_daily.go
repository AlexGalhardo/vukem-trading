package adapters

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

const (
	alpacaDailyPages   = 20
	alpacaDailySymbols = 100
)

// DailyBars returns the split-adjusted daily bars of several US symbols from the IEX feed in one paged
// request. Bars are keyed by the exchange date (America/New_York) at 00:00 UTC and, being IEX only, carry
// IEX volume, not the consolidated market volume. A symbol Alpaca does not know is simply absent.
func (a Alpaca) DailyBars(
	ctx context.Context,
	symbols []string,
	start, end time.Time,
) (map[string][]trading.Candle, error) {
	if len(symbols) == 0 || len(symbols) > alpacaDailySymbols {
		return nil, fmt.Errorf("between 1 and %d symbols are required", alpacaDailySymbols)
	}
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, err
	}
	params := url.Values{
		"symbols":    {strings.Join(symbols, ",")},
		"timeframe":  {"1Day"},
		"start":      {start.UTC().Format(time.RFC3339)},
		"end":        {end.UTC().Format(time.RFC3339)},
		"feed":       {"iex"},
		"adjustment": {"split"},
		"limit":      {"10000"},
	}
	out := map[string][]trading.Candle{}
	for page := range alpacaDailyPages {
		var response struct {
			Bars map[string][]struct {
				Time   time.Time `json:"t"`
				Open   d.Decimal `json:"o"`
				High   d.Decimal `json:"h"`
				Low    d.Decimal `json:"l"`
				Close  d.Decimal `json:"c"`
				Volume d.Decimal `json:"v"`
			} `json:"bars"`
			Next string `json:"next_page_token"`
		}
		if err := a.request(ctx, "GET", "/v2/stocks/bars?"+params.Encode(), nil, &response, true); err != nil {
			return nil, err
		}
		for symbol, bars := range response.Bars {
			for _, b := range bars {
				ny := b.Time.In(zone)
				day := time.Date(ny.Year(), ny.Month(), ny.Day(), 0, 0, 0, 0, time.UTC)
				c := trading.Candle{Time: day, Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume}
				if err := c.Validate(); err != nil {
					return nil, fmt.Errorf("%s %s: %w", symbol, day.Format(time.DateOnly), err)
				}
				if n := len(out[symbol]); n > 0 && !day.After(out[symbol][n-1].Time) {
					return nil, fmt.Errorf("%s: daily bars duplicated or out of order", symbol)
				}
				out[symbol] = append(out[symbol], c)
			}
		}
		if response.Next == "" {
			return out, nil
		}
		if page == alpacaDailyPages-1 {
			break
		}
		params.Set("page_token", response.Next)
	}
	return nil, errors.New("daily history exceeds the page limit; select a shorter period")
}

// SubmitWithStop sends a GTC limit entry whose protective stop-loss leg is attached (an OTO order), so
// the entry never exists without its stop. Whole shares only.
func (a Alpaca) SubmitWithStop(ctx context.Context, o trading.Order, stop d.Decimal) (BrokerOrder, error) {
	var out BrokerOrder
	if o.Side != "buy" {
		return out, errors.New("a stop-protected entry must be a buy")
	}
	if !stop.IsPositive() || !o.Limit.IsPositive() || !stop.LessThan(o.Limit) {
		return out, errors.New("the stop must be positive and below the entry limit")
	}
	if !o.Qty.IsInteger() || !o.Qty.IsPositive() {
		return out, errors.New("quantity must be a whole number of shares")
	}
	payload := map[string]any{
		"symbol":          o.Symbol,
		"qty":             o.Qty.String(),
		"side":            "buy",
		"type":            "limit",
		"limit_price":     o.Limit.StringFixed(2),
		"time_in_force":   "gtc",
		"order_class":     "oto",
		"stop_loss":       map[string]string{"stop_price": stop.StringFixed(2)},
		"client_order_id": o.ID,
	}
	err := a.request(ctx, "POST", "/v2/orders", payload, &out, false)
	return out, err
}
