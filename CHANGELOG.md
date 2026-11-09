# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **site:** Doc pages are dated by the author date, so a rebase merge no longer turns the `Last updated` check red; the `.githooks/pre-push` hook refreshes `site/content/last-updated.json` before a push.
- MCP tool errors return a generic message for internal failures and log the original (security audit TR-AUDIT-003).
- `POST /mcp` is throttled per client address (60 requests per minute) and `MCP_TOKEN` shorter than 32 characters is a boot error (security audit TR-AUDIT-002).
- Ticket orders on the US broker account are rejected when the latest quote is older than `Policy.MaxAgeSeconds` (security audit TR-AUDIT-001).
- `Engine.Run` treats an error raised while its context is being cancelled as a clean stop; this removes the flaky `TestPostgresWorkerFailover`.
- `State.Reserve` rejects an entry without a valid stop (`stop required`), even when a strategy bypassed `Strategy.Validate`.

### Added

- Chart indicators (SMA 20, EMA 50, Bollinger 20, session VWAP, volume, RSI 14) on history charts, with `bun run test:unit` unit tests.
- Playwright coverage for anonymous access, private API 401s, the session cookie and the indicator toggles (`e2e/access.spec.ts`).
- Vendored Cloudflare `security-audit` skill and `Security audit` workflow.
- `GET /api/stream`: Server-Sent Events channel that tells the terminal when data changed (ADR-0022).
- `adapters.BinanceTestnet`: signed, idempotent testnet-only order adapter, not wired into the engine.
- `trading.AvellanedaStoikov`: market-making quote math, not connected to any venue.
- `trading.EvaluateLiveGate`: environment-only live switch with first-period ceilings; the web service never enables live (ADR-0021, proposed).
- `POST /api/strategies/{id}/decision`: keep-or-disable verdict with a reason in the audit trail.
- Benchmarks, restore timing and `docs/runbooks/capacity.md`.

- Fuzz tests for the CSV importer and the strategy signals, run briefly in CI.
- Operations: `GET /api/metrics`, an `Uptime` workflow, runbooks (incident, restore, key rotation), a restore rehearsal script with its recorded run, a dependency policy and `bun audit` in CI.
- A read-only MCP endpoint behind `MCP_TOKEN` with a fixed tool list, and strategy drafts that need explicit owner approval; ADR-0020.
- Terminal: an always-visible safety strip, a command palette with Alt+digit page shortcuts, a server-side workspace (watchlist, price alerts, order templates), one-key paper order tickets sized by cash share and risk budget, and a canvas tape and ladder.
- Engine: price alerts evaluated on every tick and sent once through Telegram.
- TypeScript wire types generated from the Go structs, with a test that fails on drift; ADR-0019.
- Data providers: a `Provider` interface and registry with fail-fast credentials, a Binance adapter for crypto candles and the Alpaca IEX adapter behind the same interface.
- Crypto (BTCUSDT, ETHUSDT) and spot forex (EURUSD, USDBRL) for backtests under a `research` account, with 24/7 and Monday-to-Friday sessions and tick-accurate price rounding.
- A market coverage matrix in the snapshot and Settings; ADR-0018.
- Strategies: opening-range breakout, VWAP reversion, ADX-filtered trend and a DCA grid with capped inventory, declared in the parameter schema.
- Backtests: resting limit orders and partial fills over several bars, train/test/validation partitions (validation once per strategy version), walk-forward windows, Sharpe, Sortino, Calmar, SQN, block bootstrap, cost stress and deflated Sharpe, shown in the report.
- Strategy parameters declared as data and served by `GET /api/strategy-kinds`; the strategy form renders from it.
- ADR-0015 (Bun, Biome), ADR-0016 (fill rules, validation protocol), ADR-0017 (documentation site).
- Risk protections: stop-loss streak lock, per-symbol cooldown and equity-drawdown halt, configurable in the risk form.
- Close reason on every exit order; audit entries `risk.halt` and `risk.resume`; one-shot alerts for halts and stuck unknown orders.
- The daily-loss halt lifts at the next market day.

### Changed

- Require vukem v0.39.0.
- **docs:** Agent rules now cover the shared framework feedback file (`TR-` ids), the Astro docs site and `last-updated`, the security-audit skill in guidance mode, and Railway service, volume and domain rules (`.agents/rules/railway.md`).
- Vukem 0.38.0.
- Production boot fails unless `APP_URL` is an https origin without a path (`accounts.ValidateAppURL`); env templates and the deploy checklist target `https://app.trading.vukem.dev`.
- Dynamic responses carry `Cache-Control: private, no-store`; hashed assets stay immutable.
- Container images are pinned by digest in the Dockerfiles, Compose and CI.
- Backtest partitions are now train (first 60%), test (next 20%) and validation (last 20%); the old test partition was the last 30%.
- Orders that the open does not reach rest instead of being canceled; a buy never prints above the bar's high.
- Remaining `^` ranges in `package.json` pinned to exact versions.
- Risk tests: default deny, mandatory stop, risk-per-trade sizing and the daily-loss kill switch.

### Removed

- **repo:** Unused posts scaffold from `vukem new` (controller, model, request, factory, seeder); the `posts` table stays until a later contract migration

### Security

- Override `re2js` to 2.8.7 (high advisory in the markdownlint toolchain).

## [0.1.0] - 2026-11-04

### Added

- Repository standards: `.markdownlint*.jsonc`, `.githooks/commit-msg`, `.config/cliff.toml`, `biome.json`, `CODE_OF_CONDUCT.md`, `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE.md`, issue and pull request templates, `CODEOWNERS`.
- Documentation site in `site/` (Next.js, generated from `docs/`), its Docker image, CI job and the `trading.vukem.dev` Railway service.
- `docs/trading/` knowledge base from ten reference trading projects, `docs/brainstorming.md` owner decisions, setup guides for local Docker, VPS and Railway.
- `AGENTS.md`, `CLAUDE.md`, `.agents/rules` and `.agents/skills`.
- Workflows: `scorecard`, `codeql`, `pr-title`, weekly `vuln`; `.betterleaksignore` with reviewed history findings.
- `vukem-feedback.md` (git-ignored) to collect improvement and bug feedback for the Vukem framework.

### Changed

- The git remote points to `AlexGalhardo/vukem-trading`.
- Frontend tooling moved to Bun 1.4.4: `bun check` replaces `tsc`, Biome 2 lints and formats, the Dockerfile builds assets on `oven/bun:1.4.4-alpine`.
- Documentation, UI text, messages and comments translated to English; documentation files renamed to English names.
- `golangci-lint` 2.15.0 config and CI job; Go module path and project references renamed from Scooby to `vukem-trading`.
- `main` ruleset (`.config/ruleset-protect-main.json`): no deletion or force-push, linear history, required CI checks, admin bypass.
- Release workflow publishes the matching CHANGELOG section, a GHCR image and build provenance.
- `Terminal` tables use stable row keys.

[Unreleased]: https://github.com/AlexGalhardo/vukem-trading/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/AlexGalhardo/vukem-trading/releases/tag/v0.1.0
