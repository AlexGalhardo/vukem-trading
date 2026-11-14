package adapters

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	d "github.com/shopspring/decimal"
)

const (
	binanceTestnetHost = "testnet.binance.vision"
	binanceRecvWindow  = "5000"
	binanceMaxBody     = 1 << 20
)

// BinanceTestnet places and tracks paper orders on Binance's spot testnet. It refuses every other host,
// so a live endpoint cannot be configured by mistake: the only exceptions are loopback addresses, which
// tests use. Orders are LIMIT GTC with a client order id, so a retry after a timeout asks for the order
// first instead of sending it again.
type BinanceTestnet struct {
	Key, Secret string
	BaseURL     string // optional; defaults to the testnet
	Client      *http.Client
	Now         func() time.Time
}

// TestnetAck is the part of an order response the engine needs.
type TestnetAck struct {
	OrderID       int64
	ClientOrderID string
	Status        string
	Filled        d.Decimal
	CumQuote      d.Decimal
}

func (b BinanceTestnet) base() (string, error) {
	raw := b.BaseURL
	if raw == "" {
		raw = "https://" + binanceTestnetHost
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", errors.New("invalid testnet address")
	}
	host := u.Hostname()
	loopback := host == "localhost" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
	if host != binanceTestnetHost && !loopback {
		return "", errors.New("only the Binance spot testnet is allowed")
	}
	return u.Scheme + "://" + u.Host, nil
}

func (b BinanceTestnet) sign(params url.Values) string {
	now := time.Now
	if b.Now != nil {
		now = b.Now
	}
	params.Set("timestamp", strconv.FormatInt(now().UnixMilli(), 10))
	params.Set("recvWindow", binanceRecvWindow)
	query := params.Encode()
	mac := hmac.New(sha256.New, []byte(b.Secret))
	_, _ = mac.Write([]byte(query))
	return query + "&signature=" + hex.EncodeToString(mac.Sum(nil))
}

func (b BinanceTestnet) call(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	if b.Key == "" || b.Secret == "" {
		return nil, errors.New("testnet credentials are not configured")
	}
	base, err := b.base()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path+"?"+b.sign(params), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", b.Key)
	client := b.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("testnet request failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, binanceMaxBody))
	if err != nil {
		return nil, errors.New("testnet response unreadable")
	}
	if resp.StatusCode != http.StatusOK {
		// The body is not echoed: it can repeat the signed request.
		return nil, fmt.Errorf("testnet returned HTTP %d", resp.StatusCode)
	}
	return body, nil
}

type testnetOrder struct {
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	OrigClientID  string `json:"origClientOrderId"`
	Status        string `json:"status"`
	ExecutedQty   string `json:"executedQty"`
	CumQuote      string `json:"cummulativeQuoteQty"`
}

func (o testnetOrder) ack() (TestnetAck, error) {
	filled, err := d.NewFromString(o.ExecutedQty)
	if err != nil {
		return TestnetAck{}, errors.New("testnet returned an invalid quantity")
	}
	quote, err := d.NewFromString(o.CumQuote)
	if err != nil {
		return TestnetAck{}, errors.New("testnet returned an invalid amount")
	}
	id := o.ClientOrderID
	if id == "" {
		id = o.OrigClientID
	}
	return TestnetAck{OrderID: o.OrderID, ClientOrderID: id, Status: o.Status, Filled: filled, CumQuote: quote}, nil
}

func parseAck(body []byte) (TestnetAck, error) {
	var o testnetOrder
	if err := json.Unmarshal(body, &o); err != nil {
		return TestnetAck{}, errors.New("testnet returned an unreadable order")
	}
	return o.ack()
}

// Submit sends a LIMIT GTC order. side is "BUY" or "SELL".
func (b BinanceTestnet) Submit(
	ctx context.Context,
	symbol, side string,
	qty, price d.Decimal,
	clientID string,
) (TestnetAck, error) {
	if side != "BUY" && side != "SELL" {
		return TestnetAck{}, errors.New("side must be BUY or SELL")
	}
	if !qty.IsPositive() || !price.IsPositive() || clientID == "" {
		return TestnetAck{}, errors.New("quantity, price and client order id are required")
	}
	body, err := b.call(ctx, http.MethodPost, "/api/v3/order", url.Values{
		"symbol": {symbol}, "side": {side}, "type": {"LIMIT"}, "timeInForce": {"GTC"},
		"quantity": {qty.String()}, "price": {price.String()}, "newClientOrderId": {clientID},
	})
	if err != nil {
		return TestnetAck{}, err
	}
	return parseAck(body)
}

// Lookup asks for an order by its client id; it is how a retry avoids sending an order twice.
func (b BinanceTestnet) Lookup(ctx context.Context, symbol, clientID string) (TestnetAck, error) {
	body, err := b.call(
		ctx,
		http.MethodGet,
		"/api/v3/order",
		url.Values{"symbol": {symbol}, "origClientOrderId": {clientID}},
	)
	if err != nil {
		return TestnetAck{}, err
	}
	return parseAck(body)
}

// Cancel cancels an order by its client id.
func (b BinanceTestnet) Cancel(ctx context.Context, symbol, clientID string) (TestnetAck, error) {
	body, err := b.call(
		ctx,
		http.MethodDelete,
		"/api/v3/order",
		url.Values{"symbol": {symbol}, "origClientOrderId": {clientID}},
	)
	if err != nil {
		return TestnetAck{}, err
	}
	return parseAck(body)
}

// Balances returns the free balance of every asset that has one.
func (b BinanceTestnet) Balances(ctx context.Context) (map[string]d.Decimal, error) {
	body, err := b.call(ctx, http.MethodGet, "/api/v3/account", url.Values{})
	if err != nil {
		return nil, err
	}
	var account struct {
		CanTrade bool `json:"canTrade"`
		Balances []struct {
			Asset string `json:"asset"`
			Free  string `json:"free"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(body, &account); err != nil {
		return nil, errors.New("testnet returned an unreadable account")
	}
	if !account.CanTrade {
		return nil, errors.New("the testnet account cannot trade")
	}
	out := map[string]d.Decimal{}
	for _, bal := range account.Balances {
		free, err := d.NewFromString(bal.Free)
		if err != nil {
			return nil, errors.New("testnet returned an invalid balance")
		}
		if free.IsPositive() {
			out[bal.Asset] = free
		}
	}
	return out, nil
}

// SubmitStop sends the protective leg of a long entry: a SELL STOP_LOSS_LIMIT GTC that triggers at stop
// and rests at limit (at or below the stop, so a gap still fills). The spot API has no attached-stop entry,
// so the caller places this right after the entry is acknowledged and treats an entry without one as unsafe.
func (b BinanceTestnet) SubmitStop(
	ctx context.Context,
	symbol string,
	qty, stop, limit d.Decimal,
	clientID string,
) (TestnetAck, error) {
	if !qty.IsPositive() || !stop.IsPositive() || !limit.IsPositive() || clientID == "" {
		return TestnetAck{}, errors.New("quantity, stop, limit and client order id are required")
	}
	if limit.GreaterThan(stop) {
		return TestnetAck{}, errors.New("the stop limit must not be above the stop price")
	}
	body, err := b.call(ctx, http.MethodPost, "/api/v3/order", url.Values{
		"symbol": {symbol}, "side": {"SELL"}, "type": {"STOP_LOSS_LIMIT"}, "timeInForce": {"GTC"},
		"quantity": {qty.String()}, "price": {limit.String()}, "stopPrice": {stop.String()},
		"newClientOrderId": {clientID},
	})
	if err != nil {
		return TestnetAck{}, err
	}
	return parseAck(body)
}

// SymbolFilters is the part of a pair's trading rules that sizing and pricing must respect.
type SymbolFilters struct {
	Tick, Step, MinQty, MinNotional d.Decimal
}

// Filters reads the pair's tick size, lot step, minimum quantity and minimum notional from the public
// exchange-info endpoint of the testnet.
func (b BinanceTestnet) Filters(ctx context.Context, symbol string) (SymbolFilters, error) {
	base, err := b.base()
	if err != nil {
		return SymbolFilters{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, base+"/api/v3/exchangeInfo?"+url.Values{"symbol": {symbol}}.Encode(), nil,
	)
	if err != nil {
		return SymbolFilters{}, err
	}
	client := b.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return SymbolFilters{}, errors.New("testnet request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return SymbolFilters{}, fmt.Errorf("testnet returned HTTP %d", resp.StatusCode)
	}
	var info struct {
		Symbols []struct {
			Symbol  string `json:"symbol"`
			Filters []struct {
				Type        string `json:"filterType"`
				TickSize    string `json:"tickSize"`
				StepSize    string `json:"stepSize"`
				MinQty      string `json:"minQty"`
				MinNotional string `json:"minNotional"`
			} `json:"filters"`
		} `json:"symbols"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, binanceMaxBody)).Decode(&info); err != nil {
		return SymbolFilters{}, errors.New("testnet returned an unreadable exchange info")
	}
	var out SymbolFilters
	for _, s := range info.Symbols {
		if s.Symbol != symbol {
			continue
		}
		for _, f := range s.Filters {
			switch f.Type {
			case "PRICE_FILTER":
				out.Tick, err = d.NewFromString(f.TickSize)
			case "LOT_SIZE":
				out.Step, err = d.NewFromString(f.StepSize)
				if err == nil {
					out.MinQty, err = d.NewFromString(f.MinQty)
				}
			case "NOTIONAL", "MIN_NOTIONAL":
				out.MinNotional, err = d.NewFromString(f.MinNotional)
			}
			if err != nil {
				return SymbolFilters{}, errors.New("testnet returned an invalid filter")
			}
		}
		if out.Tick.IsPositive() && out.Step.IsPositive() {
			return out, nil
		}
	}
	return SymbolFilters{}, fmt.Errorf("no trading rules for %s on the testnet", symbol)
}
