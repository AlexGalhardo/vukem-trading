package reco

import (
	"cmp"
	"math"
	"slices"
	"time"

	d "github.com/shopspring/decimal"

	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

// Criterion states the ranking rule shown next to every ranking.
const Criterion = "Score = return over the lookback window divided by the standard deviation of the daily " +
	"returns in the same window (volatility-adjusted momentum). Only closed daily bars are used. A symbol is " +
	"skipped when its history is short or stale, or its median daily dollar volume is under the floor. " +
	"The score ranks past behaviour; it does not predict returns."

// RankConfig sets the lookback, list size and eligibility filters of one market.
type RankConfig struct {
	Lookback        int       // daily returns in the window; US 21 trading days and crypto 30 days are about 30 calendar days
	TopN            int       // entries in the list
	MinDollarVolume d.Decimal // floor for the median close*volume over the window
	MaxStaleDays    int       // newest bar may be at most this many calendar days before asOf
}

// USRank is the US setting. The dollar-volume floor is only a sanity check: IEX volume is a small share
// of the consolidated market, so it is not a liquidity measure.
func USRank() RankConfig {
	return RankConfig{Lookback: 21, TopN: 20, MinDollarVolume: d.NewFromInt(1_000_000), MaxStaleDays: 4}
}

// CryptoRank is the crypto setting (volume is in base units; close*volume approximates USDT turnover).
func CryptoRank() RankConfig {
	return RankConfig{Lookback: 30, TopN: 10, MinDollarVolume: d.NewFromInt(10_000_000), MaxStaleDays: 2}
}

// Entry is one ranked symbol. Percentages are in percent units (2.5 means 2.5%).
type Entry struct {
	Rank               int       `json:"rank"`
	Symbol             string    `json:"symbol"`
	Score              float64   `json:"score"`
	ReturnPct          d.Decimal `json:"return_pct"`
	DailyVolatilityPct d.Decimal `json:"daily_volatility_pct"`
	LastClose          d.Decimal `json:"last_close"`
	LastDay            string    `json:"last_day"`
	MedianDollarVolume d.Decimal `json:"median_dollar_volume"`
}

// Skip names a symbol left out of the ranking and why.
type Skip struct {
	Symbol string `json:"symbol"`
	Reason string `json:"reason"`
}

// Ranking is the ordered list plus what was left out.
type Ranking struct {
	Top      []Entry `json:"top"`
	Eligible int     `json:"eligible"`
	Skipped  []Skip  `json:"skipped"`
}

// Rank scores every series as of the given day and returns the best TopN. It is a pure function of the
// bars: nothing after the last bar of a series is read.
func Rank(series map[string][]trading.Candle, cfg RankConfig, asOf time.Time) Ranking {
	out := Ranking{Top: []Entry{}, Skipped: []Skip{}}
	var all []Entry
	for _, symbol := range sortedKeys(series) {
		e, reason := scoreSeries(symbol, series[symbol], cfg, asOf)
		if reason != "" {
			out.Skipped = append(out.Skipped, Skip{Symbol: symbol, Reason: reason})
			continue
		}
		all = append(all, e)
	}
	slices.SortStableFunc(all, func(a, b Entry) int {
		switch {
		case a.Score > b.Score:
			return -1
		case a.Score < b.Score:
			return 1
		}
		return cmp.Compare(a.Symbol, b.Symbol)
	})
	out.Eligible = len(all)
	for i, e := range all {
		if i >= cfg.TopN {
			break
		}
		e.Rank = i + 1
		out.Top = append(out.Top, e)
	}
	return out
}

func scoreSeries(symbol string, cs []trading.Candle, cfg RankConfig, asOf time.Time) (Entry, string) {
	if len(cs) < cfg.Lookback+1 {
		return Entry{}, "history shorter than the lookback window"
	}
	last := cs[len(cs)-1]
	if asOf.Sub(last.Time) > time.Duration(cfg.MaxStaleDays)*24*time.Hour {
		return Entry{}, "latest bar is stale"
	}
	w := cs[len(cs)-cfg.Lookback-1:]
	returns := make([]float64, 0, cfg.Lookback)
	turnover := make([]d.Decimal, 0, cfg.Lookback)
	for i := 1; i < len(w); i++ {
		r, _ := w[i].Close.Div(w[i-1].Close).Sub(d.NewFromInt(1)).Float64()
		returns = append(returns, r)
		turnover = append(turnover, w[i].Close.Mul(w[i].Volume))
	}
	median := medianOf(turnover)
	if median.LessThan(cfg.MinDollarVolume) {
		return Entry{}, "median dollar volume below the floor"
	}
	_, sd := meanStd(returns)
	if sd == 0 || math.IsNaN(sd) {
		return Entry{}, "no volatility in the window"
	}
	total := w[len(w)-1].Close.Div(w[0].Close).Sub(d.NewFromInt(1))
	totalF, _ := total.Float64()
	return Entry{
		Symbol:             symbol,
		Score:              math.Round(totalF/sd*10000) / 10000,
		ReturnPct:          total.Mul(d.NewFromInt(100)).Round(2),
		DailyVolatilityPct: d.NewFromFloat(sd * 100).Round(2),
		LastClose:          last.Close,
		LastDay:            last.Time.Format(time.DateOnly),
		MedianDollarVolume: median.Round(0),
	}, ""
}

func meanStd(xs []float64) (mean, sd float64) {
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	if len(xs) < 2 {
		return mean, 0
	}
	for _, x := range xs {
		sd += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(sd / float64(len(xs)-1))
}

func medianOf(xs []d.Decimal) d.Decimal {
	s := slices.Clone(xs)
	slices.SortFunc(s, func(a, b d.Decimal) int { return a.Cmp(b) })
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return s[n/2-1].Add(s[n/2]).Div(d.NewFromInt(2))
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
