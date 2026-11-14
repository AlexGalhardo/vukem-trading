// Package reco ranks a fixed universe of US stocks and crypto pairs from free end-of-day data and runs a
// documented backtest of the same rule over the same bars. It is informational: it never sends an order.
package reco

import (
	"context"
	"time"

	"github.com/AlexGalhardo/vukem-trading/internal/trading"
)

// Disclaimer is shown on the dashboard and in every message the ranking produces.
const Disclaimer = "Informational content only. This is not investment advice or a recommendation under " +
	"CVM Resolution 20. Past performance does not guarantee future results. This system never sends real " +
	"orders; you decide and are responsible for your own trades."

// B3Note records why Brazilian stocks are not ranked.
const B3Note = "B3 is not supported: no free, reliable end-of-day source was verified " +
	"(brapi's free plan needs a token, serves one ticker per call and its terms were not verified)."

// Markets served by the ranking.
const (
	MarketUS     = "US"
	MarketCrypto = "CRYPTO"
)

// Source returns closed daily bars. Each series is ascending by day (Candle.Time is the session date at
// 00:00 UTC); symbols the source does not know are absent from the result.
type Source interface {
	Name() string
	Daily(ctx context.Context, symbols []string, from, to time.Time) (map[string][]trading.Candle, error)
}

// USUniverse is a static list of large, liquid US stocks (about the 50 largest by market value in 2026-11).
// It is not point-in-time, so historical backtests carry survivorship bias.
var USUniverse = []string{
	"AAPL", "MSFT", "NVDA", "AMZN", "GOOGL", "META", "AVGO", "TSLA", "JPM", "V",
	"LLY", "WMT", "XOM", "UNH", "MA", "COST", "ORCL", "NFLX", "HD", "PG",
	"JNJ", "BAC", "ABBV", "CVX", "KO", "MRK", "CRM", "AMD", "PEP", "ADBE",
	"TMO", "CSCO", "ACN", "LIN", "MCD", "ABT", "WFC", "DIS", "GE", "IBM",
	"CAT", "QCOM", "INTU", "TXN", "VZ", "AMGN", "PM", "NOW", "ISRG", "UBER",
}

// CryptoUniverse is a static list of large non-stablecoin USDT pairs on Binance spot.
var CryptoUniverse = []string{
	"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT", "ADAUSDT", "DOGEUSDT", "TRXUSDT", "AVAXUSDT",
	"LINKUSDT", "DOTUSDT", "LTCUSDT", "BCHUSDT", "ATOMUSDT", "UNIUSDT", "XLMUSDT", "NEARUSDT", "APTUSDT",
	"ETCUSDT", "FILUSDT",
}
