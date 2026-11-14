package commands

import (
	"context"
	"flag"
	"time"

	"github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-trading/internal/reco"
)

// RecoDaily ranks the US and crypto universes from end-of-day data, stores the report and sends it to
// Telegram. The scheduler runs it after the closes; it can also be run by hand.
type RecoDaily struct{}

func (RecoDaily) Name() string { return "reco:daily" }
func (RecoDaily) Description() string {
	return "Rank US stocks and crypto from end-of-day data (informational)"
}
func (RecoDaily) Flags(*flag.FlagSet) {}

func (RecoDaily) Run(ctx context.Context, app *vukem.App, _ []string) error {
	db, err := app.DB()
	if err != nil {
		return err
	}
	return reco.Daily(ctx, reco.DailyDeps{
		Env: app.Source.Get, Store: reco.Store{DB: db}, Log: app.Log, Pause: 1100 * time.Millisecond,
	})
}
