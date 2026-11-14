// Package paper places the daily strategy entries on broker paper environments (Alpaca paper, Binance spot
// testnet). It has no live path: the adapters refuse every host that is not a paper or testnet host.
package paper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/adapters"
	"github.com/AlexGalhardo/vukem-trading/internal/reco"
	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

// Account is what a broker reports before sizing: its equity in the quote currency and the symbols it
// already holds or has working orders for.
type Account struct {
	Equity d.Decimal
	Held   map[string]bool
}

// Entry asks for one long position sized by budget, entered near LastClose and protected by a stop.
type Entry struct {
	Symbol      string
	LastClose   d.Decimal
	Budget      d.Decimal
	StopBPS     int
	SlippageBPS int
	ClientID    string
}

// Placed is the broker acknowledgement of an entry and its stop.
type Placed struct {
	Qty, Limit, Stop d.Decimal
	OrderID          string
}

// Broker is a paper environment.
type Broker interface {
	Name() string
	Account(ctx context.Context) (Account, error)
	// Enter places the entry together with its stop. It must never leave an entry without a stop.
	Enter(ctx context.Context, e Entry) (Placed, error)
}

// Result is the outcome for one target symbol.
type Result struct {
	Symbol string
	Placed *Placed
	Skip   string // why nothing was sent
	Err    error
}

// ErrNotEnabled is returned when paper trading is not switched on.
var ErrNotEnabled = errors.New("paper trading is off: set PAPER_TRADING=true")

// Enabled reports whether the owner switched paper order placement on.
func Enabled(env func(string) string) bool { return strings.EqualFold(env("PAPER_TRADING"), "true") }

// Run enters the top symbols of a ranking that the account does not hold yet. Orders get a client id derived
// from the report date and the symbol, so running twice on the same date never doubles a position (the
// broker rejects a repeated client id). A symbol that fails is reported and the others continue.
func Run(
	ctx context.Context,
	b Broker,
	date string,
	top []reco.Entry,
	cfg reco.StrategyConfig,
) ([]Result, error) {
	if len(top) == 0 {
		return nil, errors.New("the ranking is empty")
	}
	acc, err := b.Account(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s account: %w", b.Name(), err)
	}
	if !acc.Equity.IsPositive() {
		return nil, fmt.Errorf("%s account has no equity to size positions", b.Name())
	}
	n := min(len(top), cfg.HoldTop)
	slot := acc.Equity.Div(d.NewFromInt(int64(cfg.HoldTop)))
	out := make([]Result, 0, n)
	for _, e := range top[:n] {
		r := Result{Symbol: e.Symbol}
		if acc.Held[e.Symbol] {
			r.Skip = "already held or working"
			out = append(out, r)
			continue
		}
		p, err := b.Enter(ctx, Entry{
			Symbol: e.Symbol, LastClose: e.LastClose, Budget: slot, StopBPS: cfg.StopBPS,
			SlippageBPS: cfg.SlippageBPS, ClientID: "reco-" + date + "-" + e.Symbol,
		})
		if err != nil {
			r.Err = err
		} else {
			r.Placed = &p
		}
		out = append(out, r)
	}
	return out, nil
}

// AlpacaPaper is the US paper broker.
type AlpacaPaper struct{ A adapters.Alpaca }

// Name implements Broker.
func (AlpacaPaper) Name() string { return "Alpaca paper" }

// Account implements Broker.
func (p AlpacaPaper) Account(ctx context.Context) (Account, error) {
	acct, err := p.A.Account(ctx)
	if err != nil {
		return Account{}, err
	}
	if acct.TradingBlocked {
		return Account{}, errors.New("the paper account is blocked from trading")
	}
	positions, err := p.A.Positions(ctx)
	if err != nil {
		return Account{}, err
	}
	orders, err := p.A.Orders(ctx)
	if err != nil {
		return Account{}, err
	}
	held := map[string]bool{}
	for _, pos := range positions {
		held[pos.Symbol] = true
	}
	for _, o := range orders {
		held[o.Symbol] = true
	}
	return Account{Equity: acct.Equity, Held: held}, nil
}

// Enter implements Broker: whole shares, a limit entry at most SlippageBPS above the last close, and a stop leg
// attached to the same order (OTO).
func (p AlpacaPaper) Enter(ctx context.Context, e Entry) (Placed, error) {
	tick := trading.D("0.01")
	limit := e.LastClose.Mul(d.NewFromInt(1).Add(trading.Fraction(e.SlippageBPS))).Div(tick).Floor().Mul(tick)
	// A higher stop never adds risk, so it rounds up.
	stop := e.LastClose.Mul(d.NewFromInt(1).Sub(trading.Fraction(e.StopBPS))).Div(tick).Ceil().Mul(tick)
	if !limit.IsPositive() {
		return Placed{}, errors.New("invalid price")
	}
	qty := e.Budget.Div(limit).Floor()
	if !qty.IsPositive() {
		return Placed{}, errors.New("the budget does not buy one share")
	}
	order, err := p.A.SubmitWithStop(ctx, trading.Order{
		ID: e.ClientID, Symbol: e.Symbol, Side: "buy", Qty: qty, Limit: limit,
	}, stop)
	if err != nil {
		return Placed{}, err
	}
	return Placed{Qty: qty, Limit: limit, Stop: stop, OrderID: order.ID}, nil
}

// BinancePaper is the crypto paper broker on the spot testnet. Equity is the free USDT balance only.
type BinancePaper struct{ T adapters.BinanceTestnet }

// Name implements Broker.
func (BinancePaper) Name() string { return "Binance testnet" }

// Account implements Broker.
func (p BinancePaper) Account(ctx context.Context) (Account, error) {
	bal, err := p.T.Balances(ctx)
	if err != nil {
		return Account{}, err
	}
	held := map[string]bool{}
	for asset := range bal {
		if asset != "USDT" {
			held[asset+"USDT"] = true
		}
	}
	return Account{Equity: bal["USDT"], Held: held}, nil
}

// Enter implements Broker. Spot has no attached stop and a sell stop needs the asset in the wallet, so the entry
// must fill first: it is a marketable limit, and whatever is not filled right away is cancelled. The stop then
// covers exactly the filled quantity. An entry that fills but whose stop is rejected is reported as unprotected.
func (p BinancePaper) Enter(ctx context.Context, e Entry) (Placed, error) {
	f, err := p.T.Filters(ctx, e.Symbol)
	if err != nil {
		return Placed{}, err
	}
	one := d.NewFromInt(1)
	limit := e.LastClose.Mul(one.Add(trading.Fraction(e.SlippageBPS))).Div(f.Tick).Floor().Mul(f.Tick)
	stop := e.LastClose.Mul(one.Sub(trading.Fraction(e.StopBPS))).Div(f.Tick).Ceil().Mul(f.Tick)
	stopLimit := stop.Mul(d.RequireFromString("0.995")).Div(f.Tick).Floor().Mul(f.Tick)
	if !limit.IsPositive() || !stopLimit.IsPositive() {
		return Placed{}, errors.New("invalid price")
	}
	qty := e.Budget.Div(limit).Div(f.Step).Floor().Mul(f.Step)
	if qty.LessThan(f.MinQty) || qty.Mul(limit).LessThan(f.MinNotional) {
		return Placed{}, errors.New("the budget is below the pair minimum")
	}
	ack, err := p.T.Submit(ctx, e.Symbol, "BUY", qty, limit, e.ClientID)
	if err != nil {
		return Placed{}, err
	}
	if ack.Status != "FILLED" {
		cancelled, cerr := p.T.Cancel(ctx, e.Symbol, e.ClientID)
		if cerr != nil {
			// The order may have filled between the two calls: ask before giving up.
			looked, lerr := p.T.Lookup(ctx, e.Symbol, e.ClientID)
			if lerr != nil || looked.Status != "FILLED" {
				return Placed{}, fmt.Errorf(
					"entry state unknown, reconcile before retrying: %w",
					errors.Join(cerr, lerr),
				)
			}
			cancelled = looked
		}
		ack = cancelled
	}
	if !ack.Filled.IsPositive() {
		return Placed{}, errors.New("entry not filled and cancelled; no position exists")
	}
	if _, err := p.T.SubmitStop(ctx, e.Symbol, ack.Filled, stop, stopLimit, e.ClientID+"-stop"); err != nil {
		return Placed{}, fmt.Errorf("UNPROTECTED POSITION: entry filled but the stop was rejected: %w", err)
	}
	return Placed{Qty: ack.Filled, Limit: limit, Stop: stop, OrderID: ack.ClientOrderID}, nil
}
