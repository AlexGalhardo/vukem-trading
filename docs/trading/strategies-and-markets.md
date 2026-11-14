# Strategies and markets

Hypotheses, not recommendations. Every strategy ships with entry, exit, stop, sizing, session and missing-data rules, and no result is accepted from in-sample data alone.

## Strategy families

| Family | Idea | Needs | Main risk |
| --- | --- | --- | --- |
| Trend and momentum | Moving-average crossover, channel breakout, ADX filter | Warm-up, volatility-scaled stops | Whipsaws in ranges, late entries |
| Mean reversion | Z-score, RSI, Bollinger, distance to VWAP | Stationarity check, hard stop | Trends that never revert |
| Intraday | Opening-range breakout, VWAP reclaim, gap and go | Reliable intraday data, session calendar, liquidity filter | Costs and slippage dominate small edges |
| Grid and DCA | Ladder of orders around a range, scheduled accumulation | Inventory caps, range exit | Unlimited inventory in a trend |
| Market making | Quote both sides, skew by inventory | Order-book data, low latency | Adverse selection; not viable without book data |
| Pairs and spread | Cointegrated pairs | Stationarity tests, borrow costs | Regime breaks |

Implemented: SMA crossover, Z-score reversion, range breakout, opening-range breakout, VWAP reversion, ADX-filtered trend and a DCA grid with capped inventory (spot). Market making stays out of scope until order-book data and a validated fill model exist.

## Strategy contract

- Pure function of candles and parameters; deterministic; no wall-clock access.
- Parameters declared as data (name, type, range, step, default), versioned and immutable once used.
- Emits an intent with a stop distance; the risk engine sizes it.
- Declares warm-up bars and required timeframe.

## Markets

| Market | Calendar and currency | Data | Execution | Notes |
| --- | --- | --- | --- | --- |
| B3 (equities, mini-index, mini-dollar) | B3 calendar, BRL, early close and holidays, session grid changes | Licensed intraday CSV, broker feeds | Internal replay only | Costs include brokerage, exchange fees and taxes; tax treatment is out of scope |
| US equities and ETFs | Exchange clock, USD, extended hours | Alpaca IEX (not consolidated), paid SIP; free daily bars for the ranking | Alpaca Paper (`reco:paper`, entries only) | Halts, pattern day trading rules apply to live margin accounts |
| Crypto | 24/7, UTC days, quote currency USDT | Binance public klines or CSV; daily klines for the ranking | Backtest; Binance testnet runner (`reco:paper`, entries only) | Precision and lot rules per pair; exchange-specific outages |
| Forex | 24/5 in UTC, pairs | CSV import | Backtest only | No central tape; spreads vary; leverage defaults must stay off |

Keep currencies and ledgers segregated; never sum BRL and USD without an explicit FX policy. Keep `market`, `venue`, currency and calendar separate from the symbol.

## Data sources

Prefer sources with clear terms. Public endpoints used by hobby terminals are often personal-use only; record each source's terms in [../fontes.md](../fontes.md) before relying on it. Provider interfaces ([openbb.md](openbb.md)) keep paid feeds replaceable. A coverage matrix per market (live, delayed, end of day, none) belongs in the UI ([openstock.md](openstock.md)).
