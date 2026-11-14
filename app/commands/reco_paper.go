package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-trading/internal/adapters"
	"github.com/AlexGalhardo/vukem-trading/internal/paper"
	"github.com/AlexGalhardo/vukem-trading/internal/reco"
)

// maxReportAge keeps the paper runner from trading an old ranking.
const maxReportAge = 2 * 24 * time.Hour

// RecoPaper enters the stored ranking on the broker paper environments (Alpaca paper, Binance spot
// testnet). It does nothing unless PAPER_TRADING=true, and the adapters refuse every non-paper host.
type RecoPaper struct{}

func (RecoPaper) Name() string { return "reco:paper" }
func (RecoPaper) Description() string {
	return "Enter the latest ranking on Alpaca paper and Binance testnet (paper only)"
}
func (RecoPaper) Flags(*flag.FlagSet) {}

func (RecoPaper) Run(ctx context.Context, app *vukem.App, _ []string) error {
	if !paper.Enabled(app.Source.Get) {
		app.Log.Info("paper order placement is off: set PAPER_TRADING=true")
		return nil
	}
	db, err := app.DB()
	if err != nil {
		return err
	}
	rep, ok, err := reco.Store{DB: db}.Latest(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no ranking stored yet: run reco:daily first")
	}
	day, err := time.Parse(time.DateOnly, rep.Date)
	if err != nil || time.Since(day) > maxReportAge {
		return fmt.Errorf("the latest ranking (%s) is too old to trade", rep.Date)
	}
	var failures error
	for _, m := range rep.Markets {
		var (
			b   paper.Broker
			cfg reco.StrategyConfig
		)
		switch m.Market {
		case reco.MarketUS:
			key, secret := app.Source.Get("ALPACA_API_KEY"), app.Source.Get("ALPACA_API_SECRET")
			if key == "" || secret == "" {
				app.Log.Info("US paper skipped: Alpaca credentials are not set")
				continue
			}
			b, cfg = paper.AlpacaPaper{A: adapters.Alpaca{Key: key, Secret: secret}}, reco.USStrategy()
		case reco.MarketCrypto:
			key, secret := app.Source.Get("BINANCE_TESTNET_API_KEY"), app.Source.Get("BINANCE_TESTNET_API_SECRET")
			if key == "" || secret == "" {
				app.Log.Info("crypto paper skipped: Binance testnet credentials are not set")
				continue
			}
			b, cfg = paper.BinancePaper{T: adapters.BinanceTestnet{Key: key, Secret: secret}}, reco.CryptoStrategy()
		default:
			continue
		}
		if len(m.Ranking.Top) == 0 {
			continue
		}
		results, err := paper.Run(ctx, b, rep.Date, m.Ranking.Top, cfg)
		if err != nil {
			failures = errors.Join(failures, err)
			continue
		}
		for _, r := range results {
			switch {
			case r.Err != nil:
				app.Log.Error("paper entry failed", "broker", b.Name(), "symbol", r.Symbol, "error", r.Err)
				failures = errors.Join(failures, fmt.Errorf("%s %s: %w", b.Name(), r.Symbol, r.Err))
			case r.Placed != nil:
				app.Log.Info("paper entry placed", "broker", b.Name(), "symbol", r.Symbol,
					"qty", r.Placed.Qty.String(), "limit", r.Placed.Limit.String(), "stop", r.Placed.Stop.String())
			default:
				app.Log.Info("paper entry skipped", "broker", b.Name(), "symbol", r.Symbol, "reason", r.Skip)
			}
		}
	}
	return failures
}
