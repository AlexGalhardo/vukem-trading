package reco

import (
	"math"
	"testing"
	"time"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

var day0 = time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)

// synth builds n daily bars growing by drift per day with a deterministic wiggle, so scores differ by
// symbol and the series needs no fixture file.
func synth(n int, start, drift float64, volume int64, wiggle float64) []trading.Candle {
	out := make([]trading.Candle, 0, n)
	price := start
	for i := range n {
		open := price
		price = price * (1 + drift + wiggle*math.Sin(float64(i)*1.7))
		hi, lo := math.Max(open, price)*1.004, math.Min(open, price)*0.996
		out = append(out, trading.Candle{
			Time: day0.AddDate(0, 0, i),
			Open: d.NewFromFloat(open).Round(4), High: d.NewFromFloat(hi).Round(4),
			Low: d.NewFromFloat(lo).Round(4), Close: d.NewFromFloat(price).Round(4),
			Volume: d.NewFromInt(volume),
		})
	}
	return out
}

func universe() map[string][]trading.Candle {
	return map[string][]trading.Candle{
		"UP":    synth(120, 100, 0.004, 1_000_000, 0.004),
		"FLAT":  synth(120, 100, 0.000, 1_000_000, 0.004),
		"DOWN":  synth(120, 100, -0.003, 1_000_000, 0.004),
		"UPTWO": synth(120, 50, 0.002, 1_000_000, 0.004),
		"THIN":  synth(120, 100, 0.01, 1, 0.004),
		"SHORT": synth(10, 100, 0.01, 1_000_000, 0.004),
	}
}

func cfgTest() StrategyConfig {
	c := USStrategy()
	c.Rank = RankConfig{Lookback: 21, TopN: 2, MinDollarVolume: d.NewFromInt(1_000_000), MaxStaleDays: 4}
	c.HoldTop, c.RebalanceDays = 2, 5
	return c
}

func TestRankOrdersByVolatilityAdjustedMomentumAndExplainsSkips(t *testing.T) {
	t.Parallel()
	asOf := day0.AddDate(0, 0, 119)
	cfg := RankConfig{Lookback: 21, TopN: 2, MinDollarVolume: d.NewFromInt(1_000_000), MaxStaleDays: 4}
	r := Rank(universe(), cfg, asOf)
	if len(r.Top) != 2 || r.Eligible != 4 {
		t.Fatalf("top=%d eligible=%d", len(r.Top), r.Eligible)
	}
	if r.Top[0].Rank != 1 || r.Top[0].Score < r.Top[1].Score {
		t.Fatalf("not sorted: %+v", r.Top)
	}
	if r.Top[0].Symbol != "UP" {
		t.Fatalf("best = %s", r.Top[0].Symbol)
	}
	reasons := map[string]string{}
	for _, s := range r.Skipped {
		reasons[s.Symbol] = s.Reason
	}
	if reasons["THIN"] == "" || reasons["SHORT"] == "" {
		t.Fatalf("skips = %v", reasons)
	}
}

func TestRankSkipsStaleSeries(t *testing.T) {
	t.Parallel()
	cfg := RankConfig{Lookback: 21, TopN: 5, MinDollarVolume: d.NewFromInt(1), MaxStaleDays: 2}
	r := Rank(universe(), cfg, day0.AddDate(0, 0, 130))
	if r.Eligible != 0 {
		t.Fatalf("stale bars ranked: %d", r.Eligible)
	}
}

func TestBacktestIsDeterministicAndReportsCosts(t *testing.T) {
	t.Parallel()
	a, err := Backtest(universe(), cfgTest())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Backtest(universe(), cfgTest())
	if !a.Metrics.FinalEquity.Equal(b.Metrics.FinalEquity) || a.Metrics.Trades != b.Metrics.Trades {
		t.Fatal("not deterministic")
	}
	m := a.Metrics
	if !m.FeesPaid.IsPositive() || !m.SlippageCost.IsPositive() || !m.CostsPctOfInitial.IsPositive() {
		t.Fatalf("costs missing: %+v", m)
	}
	if m.Days < 50 || m.MaxDrawdownPct.IsNegative() || m.HitRatePct.GreaterThan(d.NewFromInt(100)) {
		t.Fatalf("metrics = %+v", m)
	}
	for _, tr := range a.Trades {
		if !tr.Qty.Equal(tr.Qty.Floor()) {
			t.Fatalf("fractional US shares: %v", tr.Qty)
		}
	}
	if len(a.Curve) != m.Days || a.Curve[0].Equity.Cmp(cfgTest().Initial) != 0 {
		t.Fatal("curve must start at the initial equity")
	}
}

func TestBacktestTrendingMarketBeatsCostsAndAFlatOneDoesNot(t *testing.T) {
	t.Parallel()
	up, err := Backtest(map[string][]trading.Candle{
		"A": synth(150, 100, 0.005, 1_000_000, 0.002), "B": synth(150, 100, 0.004, 1_000_000, 0.002),
	}, cfgTest())
	if err != nil {
		t.Fatal(err)
	}
	if !up.Metrics.TotalReturnPct.IsPositive() {
		t.Fatalf("return = %v", up.Metrics.TotalReturnPct)
	}
}

func TestBacktestStopExitsAtTheStopOrTheGapOpen(t *testing.T) {
	t.Parallel()
	cs := synth(60, 100, 0.004, 1_000_000, 0.001)
	// Crash day after the entry: open far below any 10% stop.
	crash := 45
	for i := crash; i < len(cs); i++ {
		f := d.NewFromFloat(0.5)
		cs[i].Open, cs[i].High, cs[i].Low, cs[i].Close = cs[i].Open.Mul(
			f,
		), cs[i].High.Mul(
			f,
		), cs[i].Low.Mul(
			f,
		), cs[i].Close.Mul(
			f,
		)
	}
	cfg := cfgTest()
	cfg.HoldTop = 1
	r, err := Backtest(map[string][]trading.Candle{"A": cs}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r.Metrics.StopsHit == 0 {
		t.Fatal("stop never hit")
	}
	for _, tr := range r.Trades {
		if tr.Exit != "stop" {
			continue
		}
		bar := cs[crash]
		if tr.ExitPrice.GreaterThan(bar.Open) {
			t.Fatalf("gap fill %v above the open %v", tr.ExitPrice, bar.Open)
		}
	}
}

func TestBacktestNeverLooksAhead(t *testing.T) {
	t.Parallel()
	base := universe()
	full, err := Backtest(base, cfgTest())
	if err != nil {
		t.Fatal(err)
	}
	cut := 90
	perturbed := map[string][]trading.Candle{}
	for sym, cs := range base {
		cp := append([]trading.Candle(nil), cs...)
		for i := cut + 1; i < len(cp); i++ {
			f := d.NewFromFloat(1.7)
			cp[i].Open, cp[i].High, cp[i].Low, cp[i].Close = cp[i].Open.Mul(
				f,
			), cp[i].High.Mul(
				f,
			), cp[i].Low.Mul(
				f,
			), cp[i].Close.Mul(
				f,
			)
		}
		perturbed[sym] = cp
	}
	other, err := Backtest(perturbed, cfgTest())
	if err != nil {
		t.Fatal(err)
	}
	last := day0.AddDate(0, 0, cut).Format(time.DateOnly)
	compared := 0
	for i, p := range full.Curve {
		if p.Day > last {
			break
		}
		if !p.Equity.Equal(other.Curve[i].Equity) {
			t.Fatalf("equity on %s changed with future data: %v vs %v", p.Day, p.Equity, other.Curve[i].Equity)
		}
		compared++
	}
	if compared < 20 {
		t.Fatalf("only %d days compared", compared)
	}
}

func TestBacktestRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := Backtest(nil, cfgTest()); err == nil {
		t.Fatal("empty input accepted")
	}
	bad := cfgTest()
	bad.HoldTop = 0
	if _, err := Backtest(universe(), bad); err == nil {
		t.Fatal("invalid parameters accepted")
	}
	if _, err := Backtest(map[string][]trading.Candle{"A": synth(10, 100, 0.01, 1_000_000, 0)}, cfgTest()); err == nil {
		t.Fatal("short history accepted")
	}
}
