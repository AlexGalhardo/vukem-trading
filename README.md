<div align="center">

<h1><strong><a href="https://trading.vukem.dev" style="text-decoration: none" target="_blank">vukem-trading</a></strong></h1>

[![CI](https://github.com/AlexGalhardo/vukem-trading/actions/workflows/ci.yml/badge.svg)](https://github.com/AlexGalhardo/vukem-trading/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE.md)
[![Go 1.27.4](https://img.shields.io/badge/go-1.27.4-00ADD8?logo=go&logoColor=white)](go.mod)
<a href="https://github.com/AlexGalhardo/vukem-trading/releases"><img src="https://img.shields.io/github/v/release/AlexGalhardo/vukem-trading?include_prereleases&label=version" alt="Version"></a>
<a href="https://scorecard.dev/viewer/?uri=github.com/AlexGalhardo/vukem-trading"><img src="https://api.scorecard.dev/projects/github.com/AlexGalhardo/vukem-trading/badge" alt="OpenSSF Scorecard"></a>

<h3><strong>Open Source Quantitative Trading Terminal: Backtesting and Paper Trading</strong></h3>

</div>

## Introduction

vukem-trading is a personal, open source trading terminal for systematic strategies. It runs backtests and **paper trading** for B3, US equities, crypto and forex, with a risk engine that enforces stops, exposure caps and a daily-loss kill switch. It never sends real orders and makes no promise of profitability.

The application is written in Go on the [Vukem framework](https://vukem.dev), with an Inertia, React and Tailwind web terminal, PostgreSQL, and an independent engine worker. The full technical documentation is at <https://trading.vukem.dev>.

## Summary

- [Introduction](#introduction)
- [Features](#features)
- [Quick start](#quick-start)
- [Documentation](#documentation)
- [Stability](#stability)
- [Contributing](#contributing)
- [Code of conduct](#code-of-conduct)
- [Security vulnerabilities](#security-vulnerabilities)
- [License](#license)

## Features

- Private dashboard with Google-only sign-in for a single owner account.
- Versioned strategies (moving-average crossover, mean reversion, breakout), deterministic and decimal-exact.
- Backtest lab with datasets that carry hash, provenance and license; cancelable jobs; exportable reports.
- Independent engine with leader election and fencing, reservations, order intents in a transactional outbox, and reconciliation of ambiguous or partial orders.
- Durable risk blocks, audit trail and integration status.
- Paper execution through Alpaca Paper (US) and an internal replay for B3.

## Quick start

```sh
git clone https://github.com/AlexGalhardo/vukem-trading.git
cd vukem-trading
cp -n .env.example .env
vukem key:generate
docker compose up --build
```

Open <http://localhost:3000>. Setup guides: [local with Docker](docs/setup-local-with-docker.md), [VPS with Docker](docs/setup-vps-with-docker.md) and [Railway](docs/setup-in-railway.md).

Frontend tooling uses Bun 1.4.4:

```sh
bun install --frozen-lockfile
bun run typecheck   # bun check
bun run lint        # Biome
bun run lint:md     # markdownlint
go vet ./... && go test -race ./...
```

## Documentation

- [ROADMAP.md](ROADMAP.md): current state, phases and checklists.
- [docs/trading/](docs/trading/README.md): research on ten reference trading projects.
- [docs/brainstorming.md](docs/brainstorming.md): owner decisions.
- [ADRs](docs/adrs/README.md) and [SDDs](docs/sdd/README.md): design decisions and specifications.
- [AGENTS.md](AGENTS.md): guide for AI agents.

## Stability

vukem-trading follows [Semantic Versioning](https://semver.org) and tags releases as `vX.Y.Z` on `main`. Changes are listed in [CHANGELOG.md](CHANGELOG.md), which follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Release notes are the matching CHANGELOG section.

## Contributing

Thank you for considering a contribution.

- Read the design documents in `docs/` (SDDs in `docs/sdd/`, decisions in `docs/adrs/`) before changing behavior.
- Write the test first. Run the specific test, lint and typecheck that cover your change.
- Update the relevant SDD, `CHANGELOG.md` (under `[Unreleased]`) and the documentation in the same change when behavior, commands or conventions change.

The full guide (setup, checks, commit style) is in [CONTRIBUTING.md](CONTRIBUTING.md).

### Pull requests

1. Create a branch from `main`.
2. Make one logical change per commit.
3. Open a pull request against `main`; its title is a Conventional Commit.

`main` accepts changes only through a pull request with green required checks and linear history.

### Commits

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) in English: `type(scope): description`.

- Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`, `chore`, `style`, `revert`.
- Scopes: `engine`, `risk`, `backtest`, `adapters`, `data`, `strategy`, `web`, `site`, `auth`, `infra`, `deps`, `repo`, `docs`.

The `.githooks/commit-msg` hook checks the header locally (`git config core.hooksPath .githooks`).

## Code of conduct

To keep the community welcoming to everyone, please review and follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Security vulnerabilities

If you discover a security vulnerability, please follow the [security policy](SECURITY.md) instead of opening a public issue.

## License

vukem-trading is open source software released under the [MIT license](LICENSE.md).

Trading involves risk. This software is for research and paper trading; nothing here is investment advice.
