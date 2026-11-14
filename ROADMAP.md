# ROADMAP: trading.vukem.dev

**Primary document for onboarding and continuity across sessions and agents.**

Updated: 2026-11-04. Repository: [AlexGalhardo/vukem-trading](https://github.com/AlexGalhardo/vukem-trading). Documentation: <https://trading.vukem.dev>.

Legend: `[x]` done and verified, `[~]` partly done (the note says what is missing), `[ ]` to do. A box is checked only when its **audit assert** passes. Never check a box from a fixture or a document alone.

## 1. Resume point

The personal **paper-only terminal is implemented and published**: Vukem web, an independent Go engine, backtesting with realistic fills and overfitting checks, a risk engine with protections, seven strategy kinds, crypto and forex backtests, a data-provider interface, a read-only MCP endpoint, runbooks, and the documentation site at <https://trading.vukem.dev>. Railway runs web, worker, site and PostgreSQL, and a push to `main` deploys. CI covers Go lint, vet and tests with PostgreSQL, frontend lint, types and end-to-end tests, the site, the production image, vulnerabilities, secrets and a short fuzz run.

Owner decisions are in [docs/brainstorming.md](docs/brainstorming.md); the research behind the design is in [docs/trading/](docs/trading/README.md).

**Everything in the phases below that can be implemented in code is implemented.** What remains needs something only the owner or time can supply:

- Real Google OAuth with the final domain, Alpaca Paper credentials, and Telegram alerts configured in production (Phases 1, 4, 5).
- A licensed B3 intraday dataset and significant out-of-sample data (Phases 2, 3).
- A prolonged paper-trading observation and the keep-or-disable decision per strategy (Phase 6).
- Binance testnet and Alpaca paper accounts to validate the paper runner (11.3, 19.7), order-book data for market making (12.5), and a live feed for a push channel (13.6).
- The OpenSSF best-practices badge registration (7.11) and the live-readiness gate (Phase 17), which is a decision, not a build.

Real capital stays out of scope.

## 2. Onboarding for a new session

1. Run `git status --short --branch` and `git log -5 --oneline`; preserve existing changes.
2. Read this file, [AGENTS.md](AGENTS.md), [README.md](README.md) and [docs/brainstorming.md](docs/brainstorming.md).
3. Read the ADRs ([docs/adrs/](docs/adrs/README.md)), especially 0011 to 0014, and the SDD of the task.
4. Read the matching rules in [.agents/rules/](.agents/rules/) before editing.
5. Check the code: a checkbox is not evidence of implementation.
6. Do the smallest authorized delivery, run the focused checks, and update this file with evidence, pending items and the next action.

Do not ask for tokens in chat; use secure provider variables. Never print secrets.

## 3. Scope and decisions

| Topic | State |
| --- | --- |
| Audience | Owner only; one Google account; no public sign-up |
| Money | Paper now, live later behind gates (Phase 17); no live adapter exists |
| Markets | B3 and US equities today; crypto and forex planned; BRL and USD ledgers segregated |
| Execution | Alpaca Paper (US); internal replay (B3); simulator for the rest |
| Strategies | SMA crossover, Z-score reversion, breakout today; opening range, VWAP, ADX trend, grid and DCA planned |
| Data | CSV import; IEX through Alpaca; free APIs, broker paper accounts and paid feeds planned behind provider interfaces |
| Risk, non-negotiable | Mandatory stop on every order; daily-loss kill switch; risk per trade as a percentage of capital; exposure caps per asset and in total |
| Stack | Go 1.27.3, Vukem 0.38.0, PostgreSQL 18, React 19 and Inertia 3, Tailwind 4, Bun 1.4.4, Biome 2 |
| Infra | Railway: web, engine, PostgreSQL, plus the `site` service |
| Docs | Next.js site in `site/`, generated from the Markdown in `docs/` |

## 4. Repository map

| Path | Purpose |
| --- | --- |
| `cmd/app`, `cmd/engine`, `cmd/e2e` | Web server, paper engine worker, end-to-end fixture server |
| `app/`, `routes/`, `bootstrap/` | Vukem application code |
| `internal/trading` | Decimals, sizing, reservations, ledger, calendars, CSV, backtests, store |
| `internal/engine` | Leader election, fencing, replay, dispatch, reconciliation |
| `internal/adapters` | Alpaca paper, data and calendar; Telegram alerts |
| `internal/providers` | Market-data provider interface, registry and the Binance and Alpaca adapters |
| `resources/js` | Inertia React terminal; `components/ui` is generated shadcn/ui |
| `site/` | Documentation site (Next.js); content generated from `docs/` |
| `docs/` | Product, architecture, ADRs, SDDs, setup guides, runbooks, MCP, trading knowledge base |
| `.agents/` | Rules and skills for agents |
| `.github/workflows/` | `ci`, `release`, `scorecard`, `codeql`, `pr-title`, `vuln` |

## 5. Phases

### Phase 0: Foundation and specification

- [x] Official starter, PostgreSQL, Compose, Git and CI.
- [x] Product, architecture, data model, planning, research and onboarding documents.
- [x] Six SDDs and ADRs 0001 to 0014.
- [x] Owner decisions recorded (grilling plus the 2026-11-04 interview).

**Audit assert:** `ls docs/adrs docs/sdd` lists every file linked from `docs/adrs/README.md` and `docs/sdd/README.md`.

### Phase 1: Identity and dashboard

- [x] Google-only login, single account, persistent subject, state, nonce and PKCE.
- [x] Shared sessions and cache, rotation and revocation, CSRF, OAuth throttling.
- [x] Private `/`, `/login`, logout; no local password or sign-up surface.
- [x] Responsive sidebar, theme, charts, empty, error and loading states.
- [x] Server guards on pages and APIs; no credential in props or bundle.
- [~] Exercise real Google OAuth with the final domain and callback. Checked 2026-11-04 in production: `GET /auth/google/redirect` answers 302 to `accounts.google.com` with the client id, state, scope `openid email profile` and callback `https://scooby-web-production.up.railway.app/auth/google/callback`. **Open:** an interactive owner login to finish the round trip, and the final web domain (the callback must be registered in the Google console).

**Audit assert:** the e2e login spec passes (`bun run test:e2e`); an unauthenticated request to `/api/snapshot` returns 401.

### Phase 2: Financial data

- [x] Segregated accounts, modes and currencies; instruments with tick and lot.
- [x] Atomic CSV import with provenance, license, hash, OHLCV, offsets and a 5-minute grid.
- [x] Rejection of invalid, duplicate and out-of-order rows; gaps identified.
- [x] IEX history through the adapter; Alpaca calendar and clock.
- [x] B3 2026 calendar, holidays, special opening, session grid changes; unreviewed years fail closed.
- [ ] Receive and validate a licensed B3 intraday dataset.
- [ ] Validate IEX with the chosen account's permissions and limits.

**Audit assert:** `go test ./internal/trading -run 'Calendar|Import|Dataset'` passes; importing a sample with a duplicate row fails.

### Phase 3: Strategies and backtesting

- [x] SMA crossover, mean reversion and breakout, deterministic, with warm-up.
- [x] Immutable configurations, numbered versions, activation separate from the account.
- [x] Persistent, cancelable, recoverable jobs with timeouts.
- [x] Next eligible bar, costs, slippage, participation cap, conservative stop, gap-through.
- [x] Dataset, configuration and code identified; equity curve, benchmark, JSON artifact.
- [x] Return, cost, drawdown, turnover, trades, win and loss metrics; 70/30 partitions.
- [ ] Evaluate strategies on licensed, significant out-of-sample data.

**Audit assert:** two runs with the same dataset hash and configuration produce byte-identical reports.

### Phase 4: Risk, paper and recovery

- [x] Decimal sizing; cash, exposure, daily-loss and position limits.
- [x] Reservation plus intent and outbox in one snapshot and transaction, with an account lock.
- [x] Deduplicated fills, partial fills, costs, ledger and position P&L.
- [x] Independent worker, advisory lock, standby and `leader_epoch`.
- [x] Reconciliation before new entries; timeout without blind retry; `unknown` state.
- [x] Block, cancel and flatten are distinct and audited.
- [x] Tests for concurrent reservations, rollback, timeout, partial, dedupe and failover.
- [ ] Exercise contracts, cancels and fills on a dedicated real Alpaca Paper account.

**Audit assert:** `go test -race ./internal/engine ./internal/trading` passes against a `_test` database.

### Phase 5: Railway and operations

- [x] Dashboard on PostgreSQL snapshots with authenticated polling.
- [x] Audit trail, heartbeat and engine readiness.
- [x] Two binaries, non-root image, Compose; sequential deploy with healthcheck and migrations.
- [x] Audited owner recovery with session revocation; backup and restore rehearsed on a test database.
- [x] Railway project, services, variables and domain created; web and engine healthy.
- [x] Daily PostgreSQL backup scheduled (provider retention six days).
- [x] Deploy on push to `main` verified with the Bun Dockerfile (2026-11-04).
- [~] Confirm budget and region (owner). External monitoring is done: the `Uptime` workflow (item 14.2).
- [ ] Validate the first backup run and a restore on the provider.
- [ ] Test Google, broker and Telegram integrations in production.

**Audit assert:** `/` returns 302, `/login` 200, `/ready` 200, `/api/snapshot` without a session 401.

### Phase 6: Continuous validation

- [x] Instrumentation and reports for drawdown and cost review.
- [ ] Observe paper trading across the agreed periods; log divergences from the backtest.
- [ ] Compare out-of-sample results without optimizing on the final test set.
- [x] Measure latency, capacity, RPO and RTO: benchmarks in `internal/trading/bench_test.go`, restore timing in `scripts/restore-rehearsal.sh`, results in [docs/runbooks/capacity.md](docs/runbooks/capacity.md). Repeat on the Railway plan in use.
- [~] Record a keep-or-disable decision per strategy: `POST /api/strategies/{id}/decision` stores the verdict and reason in the audit trail (`TestStrategyDecisionIsRecordedInTheAuditTrail`). **Open:** the decisions themselves, which need the observation period above.

Phase 6 depends on time, data and configured accounts; do not check it from synthetic fixtures.

### Phase 7: Repository standards and English-only

Goal: the repository follows the ecosystem standards and everything is in English.

- [x] **7.1 Git remote** points to `AlexGalhardo/vukem-trading`. Assert: `git remote -v`.
- [x] **7.2 Community and policy files:** `CODE_OF_CONDUCT.md`, `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE.md`, issue and PR templates, `CODEOWNERS`. Assert: files exist and `bun run lint:md` passes on them.
- [x] **7.3 Tooling:** `.markdownlint*.jsonc`, `.githooks/commit-msg`, `.config/cliff.toml`, `biome.json` (v2), `.editorconfig` with tabs. Assert: `bun run lint` and `bun run lint:md` pass.
- [x] **7.4 Bun 1.4.4:** lockfile `bun.lock`, `packageManager`, Dockerfile on `oven/bun:1.4.4-alpine`, `bun check` instead of `tsc`. Assert: `bun run typecheck` prints "No type errors"; `docker build .` succeeds.
- [x] **7.5 Workflows:** `ci`, `release` (CHANGELOG-driven, GHCR image, provenance), `scorecard`, `codeql`, `pr-title`, `vuln`. Assert: YAML parses; CI green on `main`.
- [x] **7.6 Agent files:** `AGENTS.md`, `CLAUDE.md`, `.agents/rules`, `.agents/skills`. Assert: files exist and AGENTS.md links resolve.
- [x] **7.7 Documentation translated** to English with renamed files. Assert: `grep -rP "[áàâãéêíóôõúç]" docs` returns nothing.
- [x] **7.8 Code strings translated** (UI, messages, comments, e2e). Assert: the same grep over `app internal cmd resources/js e2e` returns nothing; `go test ./...` and CI e2e pass (2026-11-04).
- [x] **7.9 Rename `Scooby` to `vukem-trading`:** module path and imports, ldflags, env examples, image tag, comments and docs. Assert: `go build ./...`, `go vet ./...` and `go test ./...` pass; `grep -rI Scooby` finds only the items below.
  - Kept on purpose: the PostgreSQL fixture user, password and database `scooby` (Compose and CI) and the Compose project name, because renaming them orphans local volumes and would change the secret-scanner rule for that fixture value. Rename them together in a dedicated change if wanted.
  - Railway services are already named `web`, `worker` and `site`; the Railway domain keeps its `scooby-web-production` hostname until a custom domain replaces it.
- [x] **7.10 `.golangci.yml`** from the ecosystem template, `golangci-lint` 2.15.0, CI job `Go lint`. Assert: `golangci-lint run ./...` prints `0 issues`. Doc comments on every exported name are off on purpose (`revive` `exported`); package comments stay required.
- [~] **7.11 OpenSSF Scorecard** green. Done: `protect-main` ruleset (no deletion or force-push, linear history, required checks, admin bypass), actions pinned by SHA, container images pinned by digest, minimal permissions, SAST (CodeQL), secret scanning, fuzz tests (`FuzzImportCSV`, `FuzzSignal`, run in CI and clean over 85 thousand and 455 thousand executions locally), provenance on releases, `re2js` high advisory fixed through an override. Open: the remaining moderate and low advisories (`minimatch` under `shadcn` and `eslint`; wait for upstream or override), the OpenSSF best-practices badge (a registration only the owner can do at bestpractices.dev). Not pursued on purpose: required pull-request review (the owner pushes to `main` directly). Assert: `gh api "repos/AlexGalhardo/vukem-trading/code-scanning/alerts?tool_name=Scorecard&state=open"` lists only the items named here.
- [x] **7.12 First release `v0.1.0`:** CHANGELOG section, annotated tag, GitHub release with the CHANGELOG notes, GHCR image with provenance (workflow run 41920869847 succeeded). Assert: `gh release view v0.1.0` shows the notes.
- [x] **7.13 Pin remaining open ranges** in `package.json` to exact versions (9 dependencies pinned to the installed versions). Assert: `grep -c '"^' package.json site/package.json` prints 0 for both; `bun install --frozen-lockfile` passes.

### Phase 8: Documentation site and DNS

Goal: `trading.vukem.dev` opens the technical documentation, with the same look as vukem.dev.

- [x] **8.1 Site shell** copied from the vukem site and rebranded (`site/`). Assert: `cd site && bun run build`.
- [x] **8.2 Content sync:** `bun run sync:docs` generates `content/docs` and `navigation.ts` from `docs/`. Assert: `bun run check:mdx` passes (54 pages).
- [x] **8.3 Site Dockerfile** (`site/Dockerfile`, context repository root, pinned Bun and Node images, non-root). Assert: `docker build -f site/Dockerfile .` succeeds.
- [x] **8.4 CI job** for the site (lint, `check:mdx`, build). Assert: job green.
- [x] **8.5 Railway `site` service** deployed after green CI. Assert: its Railway URL returns 200 and `/` redirects to `/docs/introduction`.
- [x] **8.6 DNS:** `trading.vukem.dev` CNAME and Railway verification TXT created in the Vercel zone for `vukem.dev` on 2026-11-04; Railway reports the domain verified with a valid certificate. Assert: public resolvers return the Railway edge and `/` answers 307 to `/docs/introduction` over valid TLS.
- [x] **8.7 Last-updated metadata and sitemap:** `site/content/last-updated.json` refreshed from the git history and `app/sitemap.ts` serves the pages. Assert: `bun run check:last-updated` passes right after `bun run last-updated` (67 pages). Refresh it before each release; it is not a CI gate because every documentation commit would otherwise need a second commit.

### Phase 8b: Known flaky test

- [x] **8b.1 `TestPostgresWorkerFailover`** instability fixed at the root: `Engine.Run` returned the error of an operation interrupted by its own context cancellation (`commit`, `begin`, `leadership connection lost`). Cancellation is now a clean stop. Evidence: the old code failed 20 of 60 runs under `-race`; the new code passes 50 of 50 (`go test -race -count=50 -run 'Failover|CancelledContext' ./tests`), plus a deterministic test for a cancelled context.

### Phase 9: Backtest realism and validation

Reference: [docs/trading/backtesting.md](docs/trading/backtesting.md).

- [x] **9.1 Parameter declarations as data** (`internal/trading/kinds.go`: name, label, type, min, max, step, default per kind) shared by validation, `GET /api/strategy-kinds` and the strategy form. Assert: `TestStrategyKindsSchemaIsServedAndEnforced`.
- [x] **9.2 Look-ahead test:** future candles are rescaled and earlier signals, equity and fills must not change, for every strategy kind. Assert: `TestNoStrategyLooksAhead`.
- [x] **9.3 Partial fills over several bars** with the 1% participation cap, and resting-order semantics (a limit fills only when the bar trades strictly through it; a touch does not fill). Assert: `TestPartialFillSpansSeveralBars`, `TestRestingBuyFillsOnlyWhenTheBarTradesThrough`, `TestMarketableBuyNeverPaysAboveTheBarHigh`, `TestThinBarLeavesTheOrderOpen`.
- [x] **9.4 Fill-rule ADR** written: [ADR-0016](docs/adrs/0016-fill-rules-and-validation-protocol.md). Also ADR-0015 (Bun, Biome, `bun check`) and ADR-0017 (documentation site).
- [x] **9.5 Walk-forward runner:** partition `walkforward` reports four consecutive windows with bars, trades, return, drawdown and a too-few-trades flag. Shortcut: each window starts cold with its own warm-up. Assert: `TestWalkForwardReportsEveryWindow`.
- [x] **9.6 Robustness statistics:** deterministic block bootstrap of closed trades, cost stress with doubled costs and slippage, deflated Sharpe discounted by the number of strategy versions tried, and a fewer-than-30-trades flag. Shortcut: the Sharpe variance across trials comes from the estimator's standard error, not from stored trial Sharpes. Assert: `TestMonteCarloIsDeterministicAndLowTradesAreFlagged`, `TestCostStressNeverImprovesTheResult`, `TestDeflatedSharpeDiscountsTheNumberOfTrials`.
- [x] **9.7 Three-period split:** train 60%, test 20%, validation 20%; the API accepts `validation` once per strategy version (409 unless `repeat_validation`). Assert: `TestPartitionsAreTrainTestValidation`, `TestValidationPartitionIsUsedOncePerStrategyVersion`.
- [x] **9.8 Report metrics:** Sharpe and Sortino with the stated risk-free rate, Calmar, SQN, exposure, buy-and-hold baseline, shown in the report dialog. Assert: `TestStatsAreComputedAndStated` and the golden file `internal/trading/testdata/backtest_metrics.golden.json` (`TestBacktestMetricsGolden`; refresh with `go test ./internal/trading -run Golden -update`).

### Phase 10: Risk hardening

Reference: [docs/trading/risk-and-execution.md](docs/trading/risk-and-execution.md).

- [x] **10.1 Master arm switch and default deny:** a new account starts blocked (`Halted`), entries are rejected with the reason `account blocked`, exits stay allowed. Assert: `TestDefaultDenyRejectsEntriesButAllowsExits`. Per-venue switches arrive with a second venue.
- [x] **10.2 Mandatory stop** on every entry, enforced inside `State.Reserve` (`MinStopBPS`, reason `stop required`), not only by `Strategy.Validate`. Assert: `TestEntryWithoutStopIsRejected` (0, 9 and -5 bps) and `TestEntryCarriesItsStop`.
- [x] **10.3 Risk per trade in basis points** drives sizing; the loss at the stop, including costs and slippage, never exceeds the budget. Assert: `TestRiskPerTradeSizingInBasisPoints` at 25, 50 and 100 bps.
- [x] **10.4 Daily-loss kill switch:** blocks entries, advances the generation to fence in-flight work, allows exits, lifts at the next market day (`State.NewDay`, only for the daily-loss reason; every other halt stays until the owner clears it), and every halt and resume is audited (`risk.halt`, `risk.resume` in `Store.Mutate`). Assert: `TestDailyLossKillSwitchBlocksEntriesNotExits`, `TestNewDayLiftsOnlyTheDailyLossHalt`.
- [x] **10.5 Protections** (`Policy`: `stop_streak`, `streak_lock_mins`, `cooldown_mins`, `drawdown_bps`; zero disables, new accounts default to a 3-stop streak with a 240-minute lock and a 10% drawdown halt): stop-loss streak lock, per-symbol cooldown, equity-drawdown halt. Exposed in the risk form. Assert: `TestStopStreakLocksTheSymbolThenExpires`, `TestATargetExitResetsTheStopStreak`, `TestCooldownAppliesAfterAnyExit`, `TestEquityDrawdownHaltsTheAccount`. Shortcut: a streak counts the last fill of the closing exit, not the whole position lifecycle.
- [x] **10.6 Exit model:** every exit order carries `close_reason` (`stop`, `target`, `session_close`, `signal`, `risk`, `manual`); the engine sets it for stop, target and session exits. The time-limit barrier stays with Phase 12 strategy work. Assert: `TestEveryExitCarriesACloseReason`.
- [x] **10.7 Alarms:** `State.Watch` alerts once per condition (an order `unknown` for 10 minutes or more; any halt such as the kill switch or a stale feed) and the engine sends them through Telegram when configured. Assert: `TestWatchAlertsOncePerCondition`.

### Phase 11: Data providers

Reference: [docs/trading/openbb.md](docs/trading/openbb.md) and [docs/trading/strategies-and-markets.md](docs/trading/strategies-and-markets.md).

- [x] **11.1 Provider interface** (`internal/providers`: `Provider`, `Registry`, credentials resolved as `<PROVIDER>_<NAME>`); a missing credential fails before any network call. Assert: `TestMissingCredentialFailsBeforeAnyCall`. See [ADR-0018](docs/adrs/0018-data-providers-and-research-account.md).
- [x] **11.2 Coverage matrix** per market (data mode, execution, source, notes) in the snapshot and in Settings. Assert: `TestCoverageMatrixNamesEveryMarket` and the snapshot test (`coverage` has four rows).
- [~] **11.3 Crypto:** Binance public klines (paged, closed candles only), 24/7 UTC sessions, BTCUSDT and ETHUSDT, CSV import, and a reproducible backtest. Assert: `TestBinanceParsesPagesAndKeepsOnlyClosedCandles`, `TestCryptoCSVImportsAcrossWeekendsAndBacktestsReproducibly`. `adapters.BinanceTestnet` (HMAC-signed, idempotent by client order id, testnet host only) is built and tested against a fake server. **Not done:** wiring it into the engine and validating it against a real testnet account.
- [x] **11.4 Forex import** (EURUSD, USDBRL; backtest only; Monday to Friday UTC sessions; tick-accurate rounding). Assert: `TestFXCSVImportRespectsTheWeekend`, `TestTickRoundingFollowsTheInstrument`.
- [x] **11.5 Source terms** recorded per provider in [docs/sources.md](docs/sources.md). Assert: `TestEveryProviderIsDocumentedWithItsTerms`.

### Phase 12: Strategy expansion

- [x] **12.1 Opening-range breakout** (`opening_range`: first `fast` bars of the session set the range; buys the first close above its high, exits below its low). Assert: `TestOpeningRangeBreakout`, plus `TestNoStrategyLooksAhead` over every kind.
- [x] **12.2 VWAP reversion** (`vwap_reversion`: buys a close `threshold`% below the session VWAP, exits at the VWAP). Assert: `TestVWAPReversion`.
- [x] **12.3 ADX-filtered trend** (`adx_trend`: Wilder +DI/-DI crossover while ADX is at least `threshold`). Assert: `TestADXTrendNeedsAStrongTrend`.
- [x] **12.4 DCA grid (spot) with capped inventory** (`dca_grid`: buys dips in tranches spaced by `threshold`%, at most `fast` tranches, each sized at 1/`fast` of the risk budget, stop and target measured from the average cost). Assert: `TestDCAGridKeepsInventoryWithinTheCap`. The risk engine enforces the tranche cap and spacing, not the strategy.
- [ ] **12.5 Market making** (deferred by decision, see [ADR-0016](docs/adrs/0016-fill-rules-and-validation-protocol.md)): the Avellaneda-Stoikov quote math exists (`trading.AvellanedaStoikov`, tested). Running it needs order-book data and a validated queue model, neither of which exists.

### Phase 13: Terminal UX

Reference: [docs/trading/terminal-and-agents.md](docs/trading/terminal-and-agents.md).

- [x] **13.1 Safety bar** on every page: armed or blocked with the reason, daily loss, drawdown, exposure and open orders against their limits, and a kill-switch button. Assert: the e2e test `the safety bar states the account status and ticket hotkeys call the risk engine`.
- [x] **13.2 Command palette** (Ctrl or Cmd plus K) and Alt plus 1 to 9 page shortcuts. Assert: the e2e test `command palette and numbered shortcuts move between pages`.
- [x] **13.3 Saved workspace** per owner on the server (account, ticket symbol, watchlist, alerts, order templates), see [ADR-0019](docs/adrs/0019-workspace-ticket-and-wire-types.md). The terminal is page-based, so the workspace holds what the owner changes rather than a widget grid. Assert: the e2e test `the workspace is saved on the server and survives a reload`, `TestPreferencesPersistAndAreValidated`.
- [x] **13.4 Watchlist** capped at 20 symbols, with the last close and the data mode of each market from the coverage matrix. Assert: `TestPreferencesRejectBadWorkspaces` (the cap) and the Market data page.
- [x] **13.5 Paper hotkey order templates** (`b`, `shift+b`, `x` by default) sized by a share of cash and still capped by the risk budget. Assert: `TestTicketNeverExceedsTheCashShareOrTheRiskBudget`, `TestTicketPlacesAPaperOrderThroughTheRiskEngine`; the e2e hotkey test stubs the ticket request, because the e2e server has no engine worker to arm an account.
- [~] **13.6 Canvas tape and ladder** (`components/TapeLadder.tsx`): drawn from an effect, never from React state. `GET /api/stream` (SSE, [ADR-0022](docs/adrs/0022-stream-testnet-adapter-and-quotes.md)) now tells the page when anything changed, so it refetches at once; the poll stays as fallback. Tick-level repaint still needs a live feed.
- [x] **13.7 Generated TypeScript wire types** from the Go structs (`internal/trading/wire.go` to `resources/js/gen/wire.ts`). Assert: `TestWireTypesAreCurrent` fails when the file is stale; regenerate with `go test ./internal/trading -run Wire -update`. A Go change that breaks the UI also fails `bun check`.
- [x] **13.8 Alerts:** price thresholds stored in the workspace, checked by the engine on every tick (15 seconds) and delivered through the Telegram path, each exactly once. Assert: `TestAlertsFireOnceWhenThePriceCrosses`, `TestEngineFiresAPriceAlertOnceAndPersistsIt`.

### Phase 14: Observability and operations

- [x] **14.1 Metrics:** `GET /api/metrics` (owner session) with heartbeat age, reconciliation age, orders in `unknown`, risk rejection rate and observed against expected slippage per account. Assert: `TestOperationalMetricsReportAgesRatesAndSlippage`, `TestMetricsEndpointIsAuthenticated`.
- [x] **14.2 External uptime check:** the `Uptime` workflow checks the site and the web readiness endpoint every 30 minutes and notifies through GitHub and, when its secrets exist, Telegram. Evidence (2026-11-04): a manual run succeeded (run 41932433449) and a run with a broken URL failed as intended (run 41932438383).
- [x] **14.3 Runbooks** for incident, restore and key rotation in [docs/runbooks/](docs/runbooks/README.md), plus `scripts/restore-rehearsal.sh`. Assert: the rehearsal log [restore-rehearsal-2026-11-04.md](docs/runbooks/restore-rehearsal-2026-11-04.md) shows 17 tables restored with identical row counts.
- [x] **14.4 Dependency policy:** [docs/dependency-policy.md](docs/dependency-policy.md); Dependabot covers `gomod`, `github-actions`, `docker` and `bun` for the app and `site/`; `bun audit --audit-level=high` runs in CI for both; `govulncheck`, CodeQL, betterleaks and Scorecard already run. Assert: CI green on `main`.

### Phase 15: Agents and MCP (read-only)

- [x] **15.1 Read-only MCP server** (`POST /mcp`, JSON-RPC 2.0) behind a Bearer token (`MCP_TOKEN`; off when unset) with a fixed six-tool allowlist, documented in [docs/mcp.md](docs/mcp.md). Assert: `TestMCPIsOffWithoutAToken`, `TestMCPRejectsMissingOrWrongTokens`.
- [x] **15.2 No order tools** are exposed to any agent. Assert: `TestMCPOffersOnlyReadOnlyTools` pins the tool list, rejects state-changing names and checks that order and control tool names are refused.
- [x] **15.3 Strategy drafts:** a configuration written outside the owner's session goes through `POST /api/strategies/draft`, is validated like any strategy and stays a draft that cannot be enabled until the owner approves it (audited as `strategy.draft.approved`). No language model is called by the server. Assert: `TestDraftStrategyNeedsExplicitApprovalToEnable`.

### Phase 16: Multi-user (explicitly out of scope)

Not planned. Reopen only with a new ADR covering isolation, authorization, billing and regulation.

### Phase 17: Live-readiness gate (decision only)

No live adapter is built in this roadmap. Writing an ADR is the only allowed work until all of these hold:

- [ ] Paper results over an agreed period match backtest expectations within a stated tolerance.
- [x] Risk gate, kill switch and reconciliation have failure-path tests in CI: default deny, mandatory stop, daily-loss kill switch, protections, concurrent reservations, rollback, timeouts, partial fills and leader failover (`internal/trading`, `internal/engine`, `tests`). This item alone does not open the gate.
- [x] Live is enabled by a server-side flag separate from any agent or UI setting: `trading.EvaluateLiveGate` reads only the environment (`TestLiveNeedsEveryCondition`, `TestLiveGateReadsNothingButTheEnvironment`).
- [~] Credentials encrypted at rest and present only in the worker: the gate refuses any role but `worker` (`TestOnlyTheWorkerCanEnableLive`). **Open:** secret storage is Railway variables; no live credentials exist.
- [x] First-period limits (order size, daily loss) set deliberately low: ceilings of 100 per order and 25 bps per day in code, enforced by the gate.
- [ ] Owner sign-off recorded in an ADR: [ADR-0021](docs/adrs/0021-live-readiness-gate.md) is proposed and awaits the owner.

### Phase 18: Delivery for app.trading.vukem.dev (2026-11)

- [x] **18.1 Vukem 0.38.0** (drivers on their latest compatible tags). Assert: `go build ./...`, `go vet ./...`, focused `go test`.
- [x] **18.2 Security audit skill** vendored in `.agents/skills/security-audit` with the `Security audit` workflow; referenced from [secure-coding.md](.agents/rules/secure-coding.md).
- [x] **18.3 Chart indicators:** SMA 20, EMA 50, Bollinger 20, session VWAP, volume and RSI 14 on every history chart (`resources/js/lib/indicators.ts`). Assert: `bun run test:unit`; the e2e test `a signed-in owner can chart imported history with indicators`.
- [x] **18.4 Access e2e:** anonymous page redirects, 401 on every private API (13 writes, 8 reads), HttpOnly session cookie, authenticated API reads. Assert: `bunx playwright test e2e/access.spec.ts`.
- [x] **18.5 Canonical host:** production boot fails unless `APP_URL` is an https origin (`TestValidateAppURL`); the Google `redirect_uri` is pinned by `TestGoogleRedirectUsesTheCanonicalAppURL`; env templates and the [deploy checklist](docs/deploy-checklist.md) name every variable for `https://app.trading.vukem.dev`.
- [x] **18.6 Caching:** hashed assets stay immutable and gzip (347.9 kB to 110.7 kB for the React runtime chunk); dynamic responses are `private, no-store` (`TestPrivateResponsesAreNeverCached`). Lookup columns were already indexed.
- [ ] **18.7 Owner gate:** add the `app.trading.vukem.dev` domain and DNS on Railway, set `APP_URL`, add the Google redirect URI, then run the verification list in the deploy checklist.

### Phase 19: Daily ranking, EOD backtest and paper runner (2026-11)

Owner decision 8.d: **US equities and crypto**. Design: [ADR-0023](docs/adrs/0023-daily-ranking-and-paper-runner.md), [SDD-07](docs/sdd/07-daily-ranking-and-paper-runner.md).

- [x] **19.1 Daily ranking (8.b):** top 20 US stocks and top 10 crypto over about 30 days from free end-of-day data (Alpaca IEX daily with a free key, Binance public klines), volatility-adjusted momentum, a stored report per date. Assert: `go test ./internal/reco`, `TestRecoDailyStoresSendsAndServesTheReport`.
- [x] **19.2 Job, dashboard and Telegram:** `reco:daily` at 00:30 UTC, the `/recommendations` page, one Telegram message per market (off without `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID`). Assert: `TestRecoDailyWithoutTelegramAndWithOneMarketDown`, `bun run build`.
- [x] **19.3 Disclaimer (CVM Resolution 20):** on the dashboard, in the API and in every Telegram message. Assert: `TestEveryMessageStartsAndEndsWithTheDisclaimer`, `TestRecoEndpointIsPrivateAndCarriesTheDisclaimerWhenEmpty`.
- [x] **19.4 Backtest (8.c):** one momentum rotation with stop, fees and slippage; return, benchmark, drawdown, Sharpe, hit rate, costs; deterministic and free of look-ahead. Assert: `TestBacktestNeverLooksAhead`, `TestBacktestIsDeterministicAndReportsCosts`.
- [x] **19.5 Paper runner (8.c):** `reco:paper` on Alpaca paper (OTO stop) and Binance testnet (fill, then stop-limit), off unless `PAPER_TRADING=true`, live hosts refused. Assert: `TestMoney_NoLiveEndpointCanBeUsedByThePaperBrokers`, `TestMoney_AlpacaRefusesLiveAndForeignHostsBeforeAnyRequest`.
- [ ] **19.6 B3 end-of-day:** not supported; needs a free source with verified terms (brapi free needs a token, one ticker per call).
- [ ] **19.7 Paper exits and reconciliation:** `reco:paper` only enters; rotation exits, working-order cleanup and a run against real Alpaca paper and Binance testnet accounts are open (owner credentials).
- [ ] **19.8 Legal review:** a lawyer's opinion on Resolution CVM 20 before the ranking is shared with anyone else or charged for.

#### Owner gates still open (nothing here can be faked in code)

| Gate | Needs |
| --- | --- |
| 2.x, 3.x licensed B3 intraday dataset and out-of-sample evaluation | Licensed data |
| 4.x Alpaca Paper contracts, cancels and fills; 5.x Google, broker and Telegram in production | Real accounts and production secrets |
| 6.x paper observation and keep-or-disable decisions | Time |
| 11.3, 19.7 crypto testnet and paper runner validation, 12.5 market making, 13.6 live push channel | Binance testnet account, order-book data, live feed |
| 7.11 OpenSSF badge; 17 live-readiness sign-off (ADR-0021) | Owner decision |
| `ANTHROPIC_API_KEY` repository secret for the agent half of the `Security audit` workflow | Owner secret |

## 6. Contracts and limitations

- Financial persistence uses versioned decimal JSON in a PostgreSQL account with lock and compare-and-swap plus audit in the same commit; do not claim normalized `NUMERIC` tables that do not exist. ADR-0012 explains the trade-offs and the migration path for high volume.
- The outbox is the durable list of orders approved in the snapshot. There is no external exactly-once; an in-flight request needs reconciliation even with generation and fencing.
- Web polling is 10 s, the execution loop 15 s, candles 5 m. This is not HFT, and stops are not guaranteed broker-native stops.
- The B3 calendar supports 2026 only. Extend a year only with a reviewed calendar and tests. IEX is not the consolidated market.
- The financial snapshot grows with history. Never truncate the ledger to save space; normalize or archive with a trail first.
- One active configuration per US asset and one active B3 replay per account.
- B3 is an internal historical replay, not a broker or a live feed.
- The web may use paper keys to import IEX data; all values stay on the server. Never place credentials in `VITE_*` variables, props, logs or Git.
- Push to `main` deploys to Railway. Use pull requests with green CI.
