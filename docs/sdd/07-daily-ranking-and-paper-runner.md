# SDD-07: daily ranking, backtest and paper runner

Status: implemented on 2026-11-14. Decision: [ADR-0023](../adrs/0023-daily-ranking-and-paper-runner.md).

## Purpose

Rank 50 large US stocks and 20 crypto pairs once a day from free end-of-day data, show the top 20 and top 10 on
the web terminal and in Telegram, backtest one documented rule over the same bars, and let the owner place that
rule's entries on broker paper environments. Informational only; never a real order.

## Components

| Piece | Code | Notes |
| --- | --- | --- |
| Sources | `internal/reco/sources.go`, `internal/adapters/alpaca_daily.go` | `AlpacaDaily` (IEX, key from env), `BinanceDaily` (public). Closed bars only. |
| Ranking | `internal/reco/score.go` | Pure function of the bars; `Criterion` text is shown with every ranking. |
| Backtest | `internal/reco/backtest.go` | Daily bars, fills at the next open, stop on every entry. |
| Run and store | `internal/reco/run.go`, `store.go`, `daily.go` | Table `recommendations` (one row per UTC date). |
| Telegram | `internal/reco/message.go`, `adapters.SendMessage` | One message per market, disclaimer first and last. |
| Job | `routes/schedule.go`, `app/commands/reco.go` | `reco:daily` at 00:30 UTC, no overlap, one server. |
| API and page | `GET /api/recommendations/latest`, `/recommendations` | Owner session required; the empty payload still has the disclaimer. |
| Paper runner | `internal/paper`, `app/commands/reco_paper.go` | `reco:paper`, off unless `PAPER_TRADING=true`. |

## Ranking contract

- Input: closed daily bars (`Candle.Time` is the session date at 00:00 UTC; US dates are exchange dates).
- Score: `(close_t / close_(t-L) - 1) / stdev(daily returns over L days)`, L = 21 (US) or 30 (crypto).
- Eligible: at least L+1 bars, newest bar within 4 (US) or 2 (crypto) calendar days, median `close * volume`
  over the window at or above 1,000,000 (US, IEX volume, sanity floor) or 10,000,000 (crypto).
- Output: top 20 US and top 10 crypto, each with return %, daily volatility %, score, last close, plus skipped
  symbols with the reason. Ties break by symbol. A failing market is reported with its error; the job fails
  only when every market fails.
- Not supported: B3 (no free reliable source, see ADR-0023).

## Strategy and backtest contract

- Long-only. After each close, rank the universe; every `RebalanceDays` hold the top `HoldTop` with equal slots
  of equity. US: 5 symbols, 5 days, stop 1000 bps, fee 2 bps, slippage 5 bps, whole shares. Crypto: 3 symbols,
  7 days, stop 1500 bps, fee 10 bps, slippage 5 bps, step 0.0001.
- Fills at the next open worse by the slippage; the fee is charged on both sides. The stop is checked against
  each daily low and a gap through it fills at the open. Stops round up; sizes round down.
- Metrics: total return, equal-weight buy-and-hold return, maximum drawdown, annualized Sharpe (252 or 365
  periods, zero risk-free rate), hit rate of closed trades net of costs, trades, stops hit, fees, slippage cost
  and costs as a percentage of capital.
- Tests: determinism, look-ahead (changing bars after day D leaves the equity curve up to D unchanged), stop gap
  fill, costs reported, invalid input.
- Limits: static universe (survivorship bias), 400-day window, one untuned parameter set, IEX prices, no
  walk-forward or deflated Sharpe. A replay, not broker execution and not a promise of profit.

## Paper runner contract

- Switch: `PAPER_TRADING=true`; credentials `ALPACA_API_KEY`/`ALPACA_API_SECRET` and
  `BINANCE_TESTNET_API_KEY`/`BINANCE_TESTNET_API_SECRET`. The latest report must be at most two days old.
- Sizing: broker equity divided by `HoldTop`; positions already held or working are skipped. Client order ids are
  `reco-<date>-<symbol>`, so a rerun is rejected by the broker instead of doubling a position.
- Alpaca paper: whole shares, GTC limit entry at most the slippage above the last close, stop leg attached (OTO).
- Binance testnet: limit and stop prices respect the pair tick, quantity respects step, minimum quantity and
  minimum notional (read from `exchangeInfo`). A spot stop needs the asset, so the entry must fill first;
  an unfilled entry is cancelled and a filled one gets a `STOP_LOSS_LIMIT` for the filled quantity. A rejected
  stop is logged as an unprotected position for the owner to close.
- Safety: Alpaca accepts only `paper-api.alpaca.markets` and `data.alpaca.markets`, Binance only
  `testnet.binance.vision` (loopback only for tests). Tests: `TestMoney_AlpacaRefusesLiveAndForeignHostsBeforeAnyRequest`,
  `TestTestnetRefusesEveryOtherHostBeforeAnyRequest`, `TestMoney_NoLiveEndpointCanBeUsedByThePaperBrokers`.

## Acceptance

`go test ./internal/reco ./internal/paper ./internal/adapters ./tests -run "Reco|Money|Alpaca|Testnet|SendMessage"`,
`bun run typecheck`, `bun run lint`, `bun run build`.
