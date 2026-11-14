# vukem-trading: guide for AI agents

vukem-trading is a personal quantitative trading terminal: backtesting and paper trading for B3, US equities, crypto and forex, written in Go on the [Vukem framework](https://vukem.dev) with an Inertia, React and PostgreSQL web terminal. It never sends real orders. Documentation is published at <https://trading.vukem.dev>.

## Start here

1. Read [ROADMAP.md](ROADMAP.md): current state, phases, checklists and the next concrete action.
2. Read [docs/brainstorming.md](docs/brainstorming.md) for owner decisions, then the relevant [ADRs](docs/adrs/README.md) and [SDDs](docs/sdd/README.md) before changing behavior.
3. Read the research in [docs/trading/](docs/trading/README.md) before designing strategies, backtests, risk or execution.
4. Read every matching rule in [.agents/rules/](.agents/rules/) before editing; read the applicable skill in [.agents/skills/](.agents/skills/) when its description fits. Rules are not loaded automatically.
5. For Railway operations follow [.agents/rules/railway.md](.agents/rules/railway.md), [.agents/skills/use-railway/SKILL.md](.agents/skills/use-railway/SKILL.md) and [docs/setup-in-railway.md](docs/setup-in-railway.md). Never deploy unless the owner asks.

## Working rules

- Everything in the repository (code, comments, docs, UI text, commit messages) is in English; chat with the owner in pt-BR.
- Treat [owner decisions](docs/brainstorming.md) and [ADRs](docs/adrs/README.md) as final. To change one, update the decision, then the SDD, then the code.
- Prefer native Vukem features. Do not add Go or JavaScript dependencies without owner approval and an ADR. Pin exact versions; never use `latest`, `next`, `canary` or prereleases.
- Use the official [Vukem documentation](https://vukem.dev/docs) for framework facts. Do not invent APIs; record gaps in the shared workspace `vukem-feedback.md` (ids `TR-<n>`) following [.agents/rules/framework-feedback.md](.agents/rules/framework-feedback.md).
- Paper only. An LLM never signs or sends orders. See [.agents/rules/money.md](.agents/rules/money.md).
- Make surgical changes. Do not weaken security defaults. Update the owning docs and `CHANGELOG.md` when behavior, commands or conventions change.
- Use Conventional Commits ([.agents/rules/commits.md](.agents/rules/commits.md)). Never rewrite history or force-push `main`.
- Apply the vendored `security-audit` skill ([.agents/skills/security-audit](.agents/skills/security-audit), Cloudflare) in guidance mode while developing; a full audit only on request. CI runs it on pull requests (`.github/workflows/security-audit.yml`).
- Never commit `go.work`, a local `replace` directive, `.env` files or credentials.

## Commands

```sh
bun install --frozen-lockfile   # frontend dependencies (Bun 1.4.4)
bun run typecheck               # bun check
bun run lint                    # Biome
bun run lint:md                 # markdownlint-cli2
bun run test:unit               # indicator unit tests (bun test)
bun run build                   # Vite assets
go vet ./...
go test -race ./...
go tool govulncheck ./...
```

Run the focused checks for the task; CI runs the full set. Git hooks: `git config core.hooksPath .githooks`.

## Project map

- `cmd/app`, `cmd/engine`, `cmd/e2e`: web server, paper engine worker, end-to-end fixture server.
- `app/`, `routes/`, `bootstrap/`: Vukem application code (accounts, controllers, providers, routes).
- `internal/trading`, `internal/engine`, `internal/adapters`: domain, execution loop and broker or data adapters. `internal/reco`: daily ranking and EOD backtest; `internal/paper`: paper entries (ADR-0023).
- `resources/js`: Inertia React terminal. `site/`: Astro documentation site (same shell and pinned versions as `vukem/site`) published at trading.vukem.dev; any change under `site/content/docs` needs `npm run last-updated` in `site/`.
- `docs/`: product, architecture, ADRs, SDDs, setup guides and the [trading knowledge base](docs/trading/README.md).
- `.agents/rules/`: path-scoped coding rules. `.agents/skills/`: project skills. `CLAUDE.md` points here.
