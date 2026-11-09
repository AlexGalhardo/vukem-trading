# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed

- **repo:** Unused posts scaffold from `vukem new` (controller, model, request, factory, seeder); the `posts` table stays until a later contract migration

## [0.3.0] - 2026-11-07

### Added

- **security:** Vendor the Cloudflare security-audit skill and workflow ([d2a70dd](https://github.com/AlexGalhardo/vukem-trading/commit/d2a70dda5409545ae82a072db29badb3adeb718d))
- **ui:** Add chart indicators (SMA, EMA, Bollinger, VWAP, volume, RSI) with unit tests ([051d805](https://github.com/AlexGalhardo/vukem-trading/commit/051d80590e520420d36e99a975d94d8d1adf0192))
- **auth:** Fail boot on a non-canonical production APP_URL and test the Google redirect URI ([148e142](https://github.com/AlexGalhardo/vukem-trading/commit/148e1427afd9bd1306939e48ee14bf375406a3f4))
- **web:** Mark private responses no-store ([8221476](https://github.com/AlexGalhardo/vukem-trading/commit/822147684130dafe5c3b1e3aa91800fedfc29660))

### Changed

- **site:** Migrate the documentation site from Next.js to Astro ([6e98c29](https://github.com/AlexGalhardo/vukem-trading/commit/6e98c292f2d21b633509232568e4fed5b233fe40))

### Fixed

- **site:** Answer / and /docs/ with real 301 redirects to the introduction ([b92c649](https://github.com/AlexGalhardo/vukem-trading/commit/b92c649ce3dabfe56d59c8f3d2f4c0b74d8ac0d6))
- **site:** Drop the inherited /pt-br redirect, which allowed an off-site redirect ([995288a](https://github.com/AlexGalhardo/vukem-trading/commit/995288a9d5aff623d87fdc8e2c663c3f8b6b83ef))
- **security:** Reject stale quotes on broker account tickets ([ba52870](https://github.com/AlexGalhardo/vukem-trading/commit/ba52870d694e66d36720d4a61001ad30aa9689cc))
- **security:** Throttle MCP and require a strong MCP_TOKEN ([49a922c](https://github.com/AlexGalhardo/vukem-trading/commit/49a922cf6d078464433d6cecba1b9f8fb6fe81d1))
- **security:** Stop MCP from returning internal error text ([a942580](https://github.com/AlexGalhardo/vukem-trading/commit/a942580ab6c566ef0fe296713c1e1e89dc3da470))
- **security:** Use errors.AsType in the MCP error path ([480fb9f](https://github.com/AlexGalhardo/vukem-trading/commit/480fb9f340558f464d920e0527ff75b50b9d2e5a))
- **site:** Date doc pages by author date so rebase merges keep them valid ([fb9fbaa](https://github.com/AlexGalhardo/vukem-trading/commit/fb9fbaa7db3e7a3f0d06b4be65411356520725ea))

## [0.2.0] - 2026-11-04

### Added

- **risk:** Add protections, exit reasons, halt audit and one-shot alerts ([6fb205f](https://github.com/AlexGalhardo/vukem-trading/commit/6fb205f338d5c1af3016d41c7bafbbc942cfa4f0))
- **backtest:** Add resting orders, partitions, walk-forward, statistics and robustness checks ([5f748e5](https://github.com/AlexGalhardo/vukem-trading/commit/5f748e5b0111b9f9594075efcacdef1a4945dd59))
- **strategy:** Add opening range, VWAP, ADX trend and capped DCA grid strategies ([88a3a6a](https://github.com/AlexGalhardo/vukem-trading/commit/88a3a6a3134f0cae15de96f55e8d881a054adff6))
- **data:** Add provider interface, crypto and FX backtests, coverage matrix and research account ([3397ba9](https://github.com/AlexGalhardo/vukem-trading/commit/3397ba971af35898b474fa6fa8293b263706c95a))
- **web:** Add safety strip, palette, saved workspace, tickets and alerts ([ed96f9b](https://github.com/AlexGalhardo/vukem-trading/commit/ed96f9b505fa991f35ebe236a349ec312bb169c8))
- **ops:** Add metrics, uptime, runbooks, read-only MCP and strategy drafts ([a4eaabc](https://github.com/AlexGalhardo/vukem-trading/commit/a4eaabc512948a01b73d8566ad8f74b3ca1f304b))
- Add change stream, testnet adapter, live gate, quotes and capacity runbook ([087904b](https://github.com/AlexGalhardo/vukem-trading/commit/087904bd1d116799a96158198dc327ead8599ebb))

### Fixed

- **deps:** Override re2js to 2.8.7 and record Scorecard status in the roadmap ([680559e](https://github.com/AlexGalhardo/vukem-trading/commit/680559e2e57e52091281f8a79991494ea37174aa))
- **engine:** Stop cleanly on context cancel and enforce a mandatory stop ([9e69fdc](https://github.com/AlexGalhardo/vukem-trading/commit/9e69fdcb7eedda112a03a097ccd83382eefe790c))

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

[Unreleased]: https://github.com/AlexGalhardo/vukem-trading/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/AlexGalhardo/vukem-trading/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/AlexGalhardo/vukem-trading/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/AlexGalhardo/vukem-trading/releases/tag/v0.1.0
