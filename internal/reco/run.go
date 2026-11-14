package reco

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

// HistoryDays is the calendar span fetched for the ranking and the backtest (one fetch serves both).
const HistoryDays = 400

// MarketReport is the ranking and backtest of one market, or the reason there is none.
type MarketReport struct {
	Market   string          `json:"market"`
	Source   string          `json:"source"`
	Currency string          `json:"currency"`
	Universe int             `json:"universe"`
	Ranking  Ranking         `json:"ranking"`
	Backtest *BacktestResult `json:"backtest,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// Report is the stored output of one daily run.
type Report struct {
	Date        string         `json:"date"`
	GeneratedAt time.Time      `json:"generated_at"`
	Disclaimer  string         `json:"disclaimer"`
	Criterion   string         `json:"criterion"`
	Strategy    string         `json:"strategy"`
	B3          string         `json:"b3"`
	Markets     []MarketReport `json:"markets"`
}

// Sources are the data feeds of a run.
type Sources struct {
	US, Crypto Source
}

// Run fetches the bars, ranks each market and backtests the strategy on the same bars. A market that
// fails is reported with its error and does not stop the other; Run fails only when every market failed.
func Run(ctx context.Context, now time.Time, src Sources) (Report, error) {
	now = now.UTC()
	rep := Report{
		Date: now.Format(time.DateOnly), GeneratedAt: now, Disclaimer: Disclaimer, Criterion: Criterion,
		Strategy: StrategyDescription, B3: B3Note,
	}
	plans := []struct {
		market, currency string
		src              Source
		symbols          []string
		rank             RankConfig
		cfg              StrategyConfig
	}{
		{MarketUS, "USD", src.US, USUniverse, USRank(), USStrategy()},
		{MarketCrypto, "USDT", src.Crypto, CryptoUniverse, CryptoRank(), CryptoStrategy()},
	}
	failed := 0
	for _, p := range plans {
		m := MarketReport{
			Market: p.market, Currency: p.currency, Universe: len(p.symbols),
			Ranking: Ranking{Top: []Entry{}, Skipped: []Skip{}},
		}
		if p.src == nil {
			m.Error = "no data source configured"
		} else {
			m.Source = p.src.Name()
			if err := fill(ctx, &m, p.src, p.symbols, p.rank, p.cfg, now); err != nil {
				m.Error = err.Error()
			}
		}
		if m.Error != "" {
			failed++
		}
		rep.Markets = append(rep.Markets, m)
	}
	if failed == len(plans) {
		return rep, errors.New("every market failed: " + rep.Markets[0].Error + "; " + rep.Markets[1].Error)
	}
	return rep, nil
}

func fill(
	ctx context.Context,
	m *MarketReport,
	src Source,
	symbols []string,
	rank RankConfig,
	cfg StrategyConfig,
	now time.Time,
) error {
	series, err := src.Daily(ctx, symbols, now.AddDate(0, 0, -HistoryDays), now)
	if err != nil {
		return fmt.Errorf("fetch daily bars: %w", err)
	}
	m.Ranking = Rank(series, rank, now)
	// The backtest is informational: its failure (short history) must not hide the ranking.
	if res, err := Backtest(trimTo(series, now), cfg); err == nil {
		m.Backtest = &res
	}
	return nil
}

// trimTo drops bars dated after the run day, so a clock mistake in a source cannot leak the future.
func trimTo(series map[string][]trading.Candle, now time.Time) map[string][]trading.Candle {
	out := make(map[string][]trading.Candle, len(series))
	for sym, cs := range series {
		n := len(cs)
		for n > 0 && cs[n-1].Time.After(now) {
			n--
		}
		out[sym] = cs[:n]
	}
	return out
}
