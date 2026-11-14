package reco

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/AlexGalhardo/vukem-trading/internal/adapters"
)

// DailyDeps are the collaborators of the scheduled job; tests replace them.
type DailyDeps struct {
	Env          func(string) string
	Store        Store
	Now          func() time.Time
	Log          *slog.Logger
	Sources      *Sources // nil builds the real sources from the environment
	TelegramBase string   // empty means the real Telegram API
	Pause        time.Duration
}

// Daily runs the ranking, stores it, and sends it to Telegram when TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are
// set; without them the sending is off and only logged. The data keys come from the environment only.
func Daily(ctx context.Context, deps DailyDeps) error {
	now := time.Now
	if deps.Now != nil {
		now = deps.Now
	}
	src := Sources{
		US:     AlpacaDaily{Key: deps.Env("ALPACA_API_KEY"), Secret: deps.Env("ALPACA_API_SECRET")},
		Crypto: BinanceDaily{},
	}
	if deps.Sources != nil {
		src = *deps.Sources
	}
	rep, err := Run(ctx, now(), src)
	if err != nil {
		return err
	}
	if err := deps.Store.Save(ctx, rep); err != nil {
		return err
	}
	deps.Log.Info("daily ranking stored", "date", rep.Date)

	token, chat := deps.Env("TELEGRAM_BOT_TOKEN"), deps.Env("TELEGRAM_CHAT_ID")
	if token == "" || chat == "" {
		deps.Log.Info("telegram sending is off: TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is not set")
		return nil
	}
	base := deps.TelegramBase
	if base == "" {
		base = "https://api.telegram.org"
	}
	var sendErr error
	for i, text := range Messages(rep) {
		if i > 0 && deps.Pause > 0 { // Telegram asks for at most one message per second per chat
			select {
			case <-ctx.Done():
				return errors.Join(sendErr, ctx.Err())
			case <-time.After(deps.Pause):
			}
		}
		if err := adapters.SendMessage(ctx, base, token, chat, text); err != nil {
			deps.Log.Error("telegram message failed", "error", err)
			sendErr = errors.Join(sendErr, err)
		}
	}
	return sendErr
}
