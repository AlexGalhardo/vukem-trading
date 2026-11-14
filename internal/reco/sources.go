package reco

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/adapters"
	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

// AlpacaDaily serves US daily bars from Alpaca's free IEX feed (credentials from the environment, never
// the repository). Prices and volume are IEX only, not the consolidated tape.
type AlpacaDaily struct {
	Key, Secret string
	DataURL     string // tests only; loopback
	Client      *http.Client
	// Now is the clock used to drop a session that is still open; it defaults to time.Now.
	Now func() time.Time
}

// Name implements Source.
func (AlpacaDaily) Name() string { return "Alpaca IEX daily bars (free key)" }

// Daily implements Source.
func (a AlpacaDaily) Daily(
	ctx context.Context,
	symbols []string,
	from, to time.Time,
) (map[string][]trading.Candle, error) {
	if a.Key == "" || a.Secret == "" {
		return nil, errors.New("ALPACA_API_KEY and ALPACA_API_SECRET are not configured")
	}
	broker := adapters.Alpaca{Key: a.Key, Secret: a.Secret, DataURL: a.DataURL, Client: a.Client}
	bars, err := broker.DailyBars(ctx, symbols, from, to)
	if err != nil {
		return nil, err
	}
	now := time.Now
	if a.Now != nil {
		now = a.Now
	}
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, err
	}
	ny := now().In(zone)
	today := time.Date(ny.Year(), ny.Month(), ny.Day(), 0, 0, 0, 0, time.UTC)
	afterClose := ny.Hour() > 16 || (ny.Hour() == 16 && ny.Minute() >= 15)
	for sym, cs := range bars {
		// The bar of a session that has not finished closing is partial: leave it out.
		if n := len(cs); n > 0 && !afterClose && !cs[n-1].Time.Before(today) {
			bars[sym] = cs[:n-1]
		}
	}
	return bars, nil
}

const (
	binanceDataHost = "https://data-api.binance.vision"
	binanceMaxBody  = 4 << 20
)

// BinanceDaily serves closed daily crypto klines (UTC days) from Binance's public market-data host. It
// needs no key; Binance market-data terms (personal, non-commercial use) apply.
type BinanceDaily struct {
	BaseURL string // tests only
	Client  *http.Client
	Now     func() time.Time
}

// Name implements Source.
func (BinanceDaily) Name() string { return "Binance public daily klines" }

// Daily implements Source.
func (b BinanceDaily) Daily(
	ctx context.Context,
	symbols []string,
	from, to time.Time,
) (map[string][]trading.Candle, error) {
	now := time.Now
	if b.Now != nil {
		now = b.Now
	}
	out := map[string][]trading.Candle{}
	for _, sym := range symbols {
		cs, err := b.klines(ctx, sym, from, to, now())
		if err != nil {
			// One missing or delisted pair must not sink the whole ranking.
			if errors.Is(err, errUnknownPair) {
				continue
			}
			return nil, fmt.Errorf("%s: %w", sym, err)
		}
		out[sym] = cs
	}
	return out, nil
}

var errUnknownPair = errors.New("pair not found")

func (b BinanceDaily) klines(ctx context.Context, symbol string, from, to, now time.Time) ([]trading.Candle, error) {
	base := b.BaseURL
	if base == "" {
		base = binanceDataHost
	}
	params := url.Values{
		"symbol": {symbol}, "interval": {"1d"}, "limit": {"1000"},
		"startTime": {strconv.FormatInt(from.UnixMilli(), 10)}, "endTime": {strconv.FormatInt(to.UnixMilli(), 10)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v3/klines?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := b.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("binance request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errUnknownPair
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance returned HTTP %d", resp.StatusCode)
	}
	var rows [][]json.RawMessage
	if err := json.NewDecoder(io.LimitReader(resp.Body, binanceMaxBody)).Decode(&rows); err != nil {
		return nil, errors.New("binance returned an unreadable response")
	}
	out := make([]trading.Candle, 0, len(rows))
	for _, row := range rows {
		c, closeMs, err := parseKline(row)
		if err != nil {
			return nil, err
		}
		if closeMs >= now.UnixMilli() {
			continue // the day still forming
		}
		if n := len(out); n > 0 && !c.Time.After(out[n-1].Time) {
			return nil, errors.New("binance klines duplicated or out of order")
		}
		out = append(out, c)
	}
	return out, nil
}

// parseKline maps [openTime, open, high, low, close, volume, closeTime, ...] to a candle at the UTC open date.
func parseKline(row []json.RawMessage) (trading.Candle, int64, error) {
	bad := errors.New("binance row is malformed")
	if len(row) < 7 {
		return trading.Candle{}, 0, bad
	}
	var openMs, closeMs int64
	if json.Unmarshal(row[0], &openMs) != nil || json.Unmarshal(row[6], &closeMs) != nil {
		return trading.Candle{}, 0, bad
	}
	var v [5]d.Decimal
	for i := range v {
		var s string
		if json.Unmarshal(row[i+1], &s) != nil {
			return trading.Candle{}, 0, bad
		}
		n, err := d.NewFromString(s)
		if err != nil {
			return trading.Candle{}, 0, bad
		}
		v[i] = n
	}
	c := trading.Candle{
		Time: time.UnixMilli(openMs).UTC(), Open: v[0], High: v[1], Low: v[2], Close: v[3], Volume: v[4],
	}
	return c, closeMs, c.Validate()
}
