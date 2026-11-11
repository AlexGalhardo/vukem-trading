# Contributing to vukem-trading

Thanks for helping. This guide covers setup, workflow and the rules every change follows.

## Setup

1. Install Go (see `go.mod`), Node.js, Docker Compose and the Vukem CLI (`go install github.com/AlexGalhardo/vukem/cmd/vukem@v0.41.0`).
2. Clone, enable the git hooks and start the stack:

   ```sh
   git clone https://github.com/AlexGalhardo/vukem-trading.git
   cd vukem-trading
   git config core.hooksPath .githooks
   cp -n .env.example .env
   npm ci
   docker compose up -d postgres
   vukem key:generate
   vukem migrate
   ```

3. Full local guides: [Docker](docs/setup-local-with-docker.md), [VPS](docs/setup-vps-with-docker.md) and [Railway](docs/setup-in-railway.md).

## Workflow

1. Pick a task from [ROADMAP.md](ROADMAP.md), or open an issue first for anything not on it.
2. Branch from `main`: `feat/<short-name>`, `fix/<short-name>`, `docs/<short-name>`.
3. Write the test first, then the code. Keep changes small and focused.
4. Run the checks that cover your change (below), then open a pull request against `main`.
5. The PR title must be a Conventional Commit. `main` accepts changes only through a pull request with green required checks and linear history.

## Checks

```sh
go test -race ./internal/... ./app/...
go vet ./...
go tool govulncheck ./...
npm run typecheck
npm run lint
npm run lint:md
```

Run the specific test, lint and typecheck that cover your change; CI runs the full suite.

## Commit messages

[Conventional Commits 1.0](https://www.conventionalcommits.org/en/v1.0.0/), enforced by the `.githooks/commit-msg` hook and by CI:

```text
feat(risk): enforce a daily-loss kill switch
fix(engine)!: reject orders without a stop

BREAKING CHANGE: strategies must now declare a stop distance.
```

Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`, `chore`, `style`, `revert`. Scope is the area (`engine`, `risk`, `backtest`, `adapters`, `web`, `site`, `infra`, `deps`, `docs`).

## Code rules

- Everything in the repository (code, comments, docs, UI text) is in English.
- Money and prices use decimal arithmetic, never floats.
- Comments explain *why*, never *what*. Fix root causes; never silence an error or skip a test.
- No new dependency without an ADR. Pin exact versions; never `latest`, `next`, `canary` or prereleases.
- No credential in code, props, logs, fixtures or docs. Broker secrets live only in the engine worker.
- Strategy and risk changes keep paper-only behavior: this project never sends real orders.

## Changelog and releases

Release notes follow [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) in [CHANGELOG.md](CHANGELOG.md): add entries under `[Unreleased]` in the same change. Versioning follows [SemVer](https://semver.org); tags are `vX.Y.Z` on `main`. A tag without its `CHANGELOG.md` section does not publish a release.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). Report security issues privately, as described in [SECURITY.md](SECURITY.md).
