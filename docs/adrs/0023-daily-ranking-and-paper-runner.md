# ADR-0023: daily ranking, end-of-day data and the paper runner

State: implemented on 2026-11-14. Details: [SDD-07](../sdd/07-daily-ranking-and-paper-runner.md). Owner decision
ROADMAP 8.d: **US equities and crypto**. Research: [web-research.md](../web-research.md) (2026-11-14).

## Context

The owner wants a daily ranking (top 20 US stocks and top 10 crypto over the last 30 days) on the web
terminal and in Telegram, plus paper trading and a local backtest of one simple strategy. Free data is the
constraint. A ranking that looks like advice has a regulatory side (CVM Resolution 20), and the project never
sends real orders.

## Decision

1. **Data, free and end of day.**
   - US: Alpaca historical **daily** bars, `feed=iex`, `adjustment=split`, one paged multi-symbol request
     (a free Alpaca paper key, read from `ALPACA_API_KEY` and `ALPACA_API_SECRET`, never the repository).
     Prices and volume are IEX only, not the consolidated tape.
   - Crypto: Binance public daily klines from `data-api.binance.vision` (no key), closed UTC days only.
   - B3: **not supported** in the ranking. brapi's free plan needs a token, serves one ticker per call and its
     terms are unverified; Stooq's CSV endpoint answers with a JavaScript challenge and is unusable from a
     server. B3 stays on CSV import and the internal replay simulator.
   - Alpha Vantage (25 requests a day) and Twelve Data (8 credits a minute) were rejected: they cannot serve a
     50-symbol universe in one run. CoinGecko Demo needs a key and adds nothing over Binance.
2. **Ranking criterion.** Score = return over the lookback window divided by the standard deviation of the
   daily returns in that window (volatility-adjusted momentum). Lookback is 21 trading days for US and 30 days
   for crypto (both about 30 calendar days). The universe is a static list of 50 large US stocks and 20 crypto
   pairs; eligibility needs a full window, a fresh last bar and a median dollar-volume floor. The score ranks
   past behaviour and is never presented as a forecast.
3. **Job.** `reco:daily` runs from the Vukem scheduler at 00:30 UTC (after the US close in every season and
   after the UTC crypto day), stores one report per date and sends one Telegram message per market. Without
   `TELEGRAM_BOT_TOKEN` or `TELEGRAM_CHAT_ID` the sending is off and logged.
4. **Disclaimer.** A fixed text (not investment advice or a recommendation under CVM Resolution 20; no
   guarantee; no orders sent; the user decides) is on the dashboard (top and bottom), in the API payload and at
   the start and end of every Telegram message. A test fails if a message lacks it.
5. **Strategy and backtest.** One long-only momentum rotation: hold the top K by score, rebalance every N days,
   fill at the next open with slippage and fees, a stop on every entry. The backtest runs over the same bars
   and reports return, an equal-weight buy-and-hold benchmark, drawdown, Sharpe, hit rate, trades, stops and
   costs. It has a look-ahead test and is deterministic.
6. **Paper runner.** `reco:paper` enters the stored ranking on Alpaca paper (OTO order: limit entry with an
   attached stop) and the Binance spot testnet (marketable limit, then a stop-limit on the filled quantity; an
   unfilled entry is cancelled). It runs only with `PAPER_TRADING=true`. The adapters accept only the paper and
   testnet hosts (loopback in tests), and `TestMoney_NoLiveEndpointCanBeUsedByThePaperBrokers` proves that live
   hosts are refused before any request.

## Alternatives

- Paid feeds (brapi paid plans, Alpaca Algo Trader Plus): better data, a cost the owner has not approved.
- Ranking inside the engine worker: couples a batch job to the order loop; a scheduled command is simpler.
- Reusing the 5-minute dataset and engine for daily bars: the intraday engine models sessions and gaps that do
  not exist in daily bars; a small separate daily backtester is clearer.

## Consequences

- The IEX volume makes the US liquidity floor a sanity check only; consolidated volume needs a paid plan.
- The universe is static, so any historical backtest has survivorship bias and the 400-day window is short.
  The result is an illustration of the rule, not evidence of an edge.
- The paper runner only enters. Rotation exits and reconciliation of working orders are not automated yet.
- Telegram sending uses the same variables as the engine alerts; the scheduler runs in the process with
  `SCHEDULE_IN_SERVE=true` (the web service by default), so that service needs the Telegram variables.

## Review trigger

A paid data plan, a B3 source with verified terms, a second strategy, any request to send to other people
(legal review first), or the first live-readiness work (ADR-0021).
