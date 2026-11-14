package reco

import (
	"errors"
	"math"
	"slices"
	"sort"
	"time"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

// StrategyDescription documents the one rule that is backtested and paper traded.
const StrategyDescription = "Long-only momentum rotation. After each close, rank the universe with the " +
	"published score; every RebalanceDays hold the top HoldTop symbols with equal slots of equity. Orders fill " +
	"at the next open, worse by the slippage, and pay the fee on both sides. Every entry carries a stop " +
	"StopBPS below its fill, checked against each daily low (a gap through the stop fills at the open). " +
	"Symbols that leave the top list are sold at the next open. Fees and slippage are assumptions, not quotes."

// StrategyConfig holds the immutable parameters of the strategy; changing one makes a new version.
type StrategyConfig struct {
	Rank           RankConfig `json:"-"`
	Lookback       int        `json:"lookback"`
	HoldTop        int        `json:"hold_top"`
	RebalanceDays  int        `json:"rebalance_days"`
	StopBPS        int        `json:"stop_bps"`
	FeeBPS         int        `json:"fee_bps"`
	SlippageBPS    int        `json:"slippage_bps"`
	Step           d.Decimal  `json:"step"`
	PeriodsPerYear int        `json:"periods_per_year"`
	Initial        d.Decimal  `json:"initial"`
}

// USStrategy is the US setting (whole shares, 2 bps fee, 5 bps slippage per side).
func USStrategy() StrategyConfig {
	r := USRank()
	return StrategyConfig{
		Rank: r, Lookback: r.Lookback, HoldTop: 5, RebalanceDays: 5, StopBPS: 1000, FeeBPS: 2, SlippageBPS: 5,
		Step: d.NewFromInt(1), PeriodsPerYear: 252, Initial: d.NewFromInt(100000),
	}
}

// CryptoStrategy is the crypto setting (10 bps spot fee, 5 bps slippage per side, 365 trading days).
func CryptoStrategy() StrategyConfig {
	r := CryptoRank()
	return StrategyConfig{
		Rank: r, Lookback: r.Lookback, HoldTop: 3, RebalanceDays: 7, StopBPS: 1500, FeeBPS: 10, SlippageBPS: 5,
		Step: d.RequireFromString("0.0001"), PeriodsPerYear: 365, Initial: d.NewFromInt(100000),
	}
}

// Trade is one closed round trip.
type Trade struct {
	Symbol     string    `json:"symbol"`
	EntryDay   string    `json:"entry_day"`
	ExitDay    string    `json:"exit_day"`
	Qty        d.Decimal `json:"qty"`
	EntryPrice d.Decimal `json:"entry_price"`
	ExitPrice  d.Decimal `json:"exit_price"`
	NetPnL     d.Decimal `json:"net_pnl"`
	Exit       string    `json:"exit"` // "rebalance" or "stop"
}

// Point is the end-of-day equity.
type Point struct {
	Day    string    `json:"day"`
	Equity d.Decimal `json:"equity"`
}

// Metrics summarize a backtest. Percentages are in percent units; money is in the market quote currency.
type Metrics struct {
	Days               int       `json:"days"`
	TotalReturnPct     d.Decimal `json:"total_return_pct"`
	BenchmarkReturnPct d.Decimal `json:"benchmark_return_pct"`
	MaxDrawdownPct     d.Decimal `json:"max_drawdown_pct"`
	Sharpe             float64   `json:"sharpe"`
	HitRatePct         d.Decimal `json:"hit_rate_pct"`
	Trades             int       `json:"trades"`
	StopsHit           int       `json:"stops_hit"`
	OpenPositions      int       `json:"open_positions"`
	FeesPaid           d.Decimal `json:"fees_paid"`
	SlippageCost       d.Decimal `json:"slippage_cost"`
	CostsPctOfInitial  d.Decimal `json:"costs_pct_of_initial"`
	FinalEquity        d.Decimal `json:"final_equity"`
}

// BacktestResult is the report of one run. It is a replay of past bars, not broker execution.
type BacktestResult struct {
	Strategy StrategyConfig `json:"strategy"`
	From     string         `json:"from"`
	To       string         `json:"to"`
	Metrics  Metrics        `json:"metrics"`
	Curve    []Point        `json:"curve"`
	Trades   []Trade        `json:"trades"`
}

type holding struct {
	qty, entry, stop, entryFee d.Decimal
	day                        time.Time
}

type buyOrder struct {
	symbol string
	slot   d.Decimal
}

// Backtest replays the strategy over the bars. Decisions on day t use only bars up to t and fill on t+1.
func Backtest(series map[string][]trading.Candle, cfg StrategyConfig) (BacktestResult, error) {
	if cfg.HoldTop < 1 || cfg.RebalanceDays < 1 || !cfg.Initial.IsPositive() || !cfg.Step.IsPositive() {
		return BacktestResult{}, errors.New("invalid strategy parameters")
	}
	days := unionDays(series)
	if len(days) == 0 {
		return BacktestResult{}, errors.New("no bars to backtest")
	}
	byDay := map[string]map[time.Time]trading.Candle{}
	for sym, cs := range series {
		m := make(map[time.Time]trading.Candle, len(cs))
		for _, c := range cs {
			m[c.Time] = c
		}
		byDay[sym] = m
	}
	feeRate, slip, stopRate := trading.Fraction(
		cfg.FeeBPS,
	), trading.Fraction(
		cfg.SlippageBPS,
	), trading.Fraction(
		cfg.StopBPS,
	)
	one := d.NewFromInt(1)

	cash := cfg.Initial
	held := map[string]*holding{}
	lastClose := map[string]d.Decimal{}
	var pendingSells []string
	var pendingBuys []buyOrder
	var curve []Point
	var trades []Trade
	fees, slipCost := trading.Zero, trading.Zero
	started, lastRebalance, stops := false, -1, 0

	closeOut := func(sym string, day time.Time, ref d.Decimal, reason string) {
		h := held[sym]
		fill := ref.Mul(one.Sub(slip))
		notional := fill.Mul(h.qty)
		fee := notional.Mul(feeRate)
		cash = cash.Add(notional).Sub(fee)
		fees = fees.Add(fee)
		slipCost = slipCost.Add(ref.Sub(fill).Mul(h.qty))
		trades = append(trades, Trade{
			Symbol: sym, EntryDay: h.day.Format(time.DateOnly), ExitDay: day.Format(time.DateOnly), Qty: h.qty,
			EntryPrice: h.entry, ExitPrice: fill,
			NetPnL: fill.Sub(h.entry).Mul(h.qty).Sub(h.entryFee).Sub(fee), Exit: reason,
		})
		delete(held, sym)
	}

	for i, day := range days {
		// 1. Fill what the previous close decided, at today's open.
		var keepSells []string
		for _, sym := range pendingSells {
			bar, ok := byDay[sym][day]
			if !ok {
				keepSells = append(keepSells, sym)
				continue
			}
			if held[sym] != nil {
				closeOut(sym, day, bar.Open, "rebalance")
			}
		}
		pendingSells = keepSells
		var keepBuys []buyOrder
		for _, b := range pendingBuys {
			bar, ok := byDay[b.symbol][day]
			if !ok {
				keepBuys = append(keepBuys, b)
				continue
			}
			fill := bar.Open.Mul(one.Add(slip))
			budget := d.Min(b.slot, cash)
			qty := budget.Div(fill.Mul(one.Add(feeRate))).Div(cfg.Step).Floor().Mul(cfg.Step)
			if !qty.IsPositive() || held[b.symbol] != nil {
				continue
			}
			notional := fill.Mul(qty)
			fee := notional.Mul(feeRate)
			cash = cash.Sub(notional).Sub(fee)
			fees = fees.Add(fee)
			slipCost = slipCost.Add(fill.Sub(bar.Open).Mul(qty))
			// A higher stop price never adds risk, so round the stop up.
			held[b.symbol] = &holding{
				qty: qty, entry: fill, stop: fill.Mul(one.Sub(stopRate)).RoundCeil(8), entryFee: fee, day: day,
			}
		}
		pendingBuys = keepBuys

		// 2. Stops against today's range.
		for _, sym := range sortedKeys(held) {
			bar, ok := byDay[sym][day]
			if !ok {
				continue
			}
			if h := held[sym]; bar.Low.LessThanOrEqual(h.stop) {
				closeOut(sym, day, d.Min(bar.Open, h.stop), "stop")
				stops++
			}
		}

		// 3. Mark to market.
		equity := cash
		for _, sym := range sortedKeys(held) {
			if bar, ok := byDay[sym][day]; ok {
				lastClose[sym] = bar.Close
			}
			equity = equity.Add(held[sym].qty.Mul(lastClose[sym]))
		}
		if started {
			curve = append(curve, Point{Day: day.Format(time.DateOnly), Equity: equity})
		}

		// 4. Decide for tomorrow using bars up to today only.
		if i+1 >= len(days) || (lastRebalance >= 0 && i-lastRebalance < cfg.RebalanceDays) {
			continue
		}
		asOf := map[string][]trading.Candle{}
		for sym, cs := range series {
			n := sort.Search(len(cs), func(k int) bool { return cs[k].Time.After(day) })
			asOf[sym] = cs[:n]
		}
		top := Rank(asOf, RankConfig{
			Lookback: cfg.Rank.Lookback, TopN: cfg.HoldTop, MinDollarVolume: cfg.Rank.MinDollarVolume,
			MaxStaleDays: cfg.Rank.MaxStaleDays,
		}, day).Top
		if len(top) == 0 {
			continue
		}
		if !started {
			started = true
			curve = append(curve, Point{Day: day.Format(time.DateOnly), Equity: equity})
		}
		lastRebalance = i
		target := map[string]bool{}
		slot := equity.Div(d.NewFromInt(int64(cfg.HoldTop)))
		pendingSells, pendingBuys = nil, nil
		for _, e := range top {
			target[e.Symbol] = true
			if held[e.Symbol] == nil {
				pendingBuys = append(pendingBuys, buyOrder{symbol: e.Symbol, slot: slot})
			}
		}
		for _, sym := range sortedKeys(held) {
			if !target[sym] {
				pendingSells = append(pendingSells, sym)
			}
		}
	}
	if len(curve) < 2 {
		return BacktestResult{}, errors.New("not enough history to rank and trade")
	}
	m := metricsOf(curve, trades, cfg)
	m.StopsHit = stops
	m.OpenPositions = len(held)
	m.FeesPaid, m.SlippageCost = fees.Round(2), slipCost.Round(2)
	m.CostsPctOfInitial = fees.Add(slipCost).Div(cfg.Initial).Mul(d.NewFromInt(100)).Round(2)
	m.BenchmarkReturnPct = benchmark(series, curve[0].Day, curve[len(curve)-1].Day)
	return BacktestResult{
		Strategy: cfg, From: curve[0].Day, To: curve[len(curve)-1].Day, Metrics: m, Curve: curve, Trades: trades,
	}, nil
}

func unionDays(series map[string][]trading.Candle) []time.Time {
	set := map[time.Time]struct{}{}
	for _, cs := range series {
		for _, c := range cs {
			set[c.Time] = struct{}{}
		}
	}
	days := make([]time.Time, 0, len(set))
	for t := range set {
		days = append(days, t)
	}
	slices.SortFunc(days, func(a, b time.Time) int { return a.Compare(b) })
	return days
}

func metricsOf(curve []Point, trades []Trade, cfg StrategyConfig) Metrics {
	hundred := d.NewFromInt(100)
	first, last := curve[0].Equity, curve[len(curve)-1].Equity
	m := Metrics{
		Days: len(curve), FinalEquity: last.Round(2),
		TotalReturnPct: last.Div(first).Sub(d.NewFromInt(1)).Mul(hundred).Round(2),
		Trades:         len(trades),
	}
	peak, worst := first, trading.Zero
	returns := make([]float64, 0, len(curve)-1)
	for i, p := range curve {
		if p.Equity.GreaterThan(peak) {
			peak = p.Equity
		}
		if dd := peak.Sub(p.Equity).Div(peak); dd.GreaterThan(worst) {
			worst = dd
		}
		if i > 0 {
			r, _ := p.Equity.Div(curve[i-1].Equity).Sub(d.NewFromInt(1)).Float64()
			returns = append(returns, r)
		}
	}
	m.MaxDrawdownPct = worst.Mul(hundred).Round(2)
	if mean, sd := meanStd(returns); sd > 0 {
		m.Sharpe = math.Round(mean/sd*math.Sqrt(float64(cfg.PeriodsPerYear))*100) / 100
	}
	wins := 0
	for _, t := range trades {
		if t.NetPnL.IsPositive() {
			wins++
		}
	}
	if len(trades) > 0 {
		m.HitRatePct = d.NewFromInt(int64(wins)).Div(d.NewFromInt(int64(len(trades)))).Mul(hundred).Round(2)
	}
	return m
}

// benchmark is the equal-weight buy-and-hold return of the symbols priced on both days, before any cost.
func benchmark(series map[string][]trading.Candle, from, to string) d.Decimal {
	sum, n := trading.Zero, int64(0)
	for _, cs := range series {
		var a, b *trading.Candle
		for k := range cs {
			switch cs[k].Time.Format(time.DateOnly) {
			case from:
				a = &cs[k]
			case to:
				b = &cs[k]
			}
		}
		if a != nil && b != nil {
			sum = sum.Add(b.Close.Div(a.Close).Sub(d.NewFromInt(1)))
			n++
		}
	}
	if n == 0 {
		return trading.Zero
	}
	return sum.Div(d.NewFromInt(n)).Mul(d.NewFromInt(100)).Round(2)
}
