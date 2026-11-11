# vukem-trading — product and architecture documentation

The [ROADMAP.md at the root](../ROADMAP.md) is the project's main document.
Read it first for onboarding, the actual state and the next action; this index
gathers the reference documents.

Data base: 2026-11-03. Document language: English.

**Current state: paper implementation delivered, external publication/configuration pending.**
The [ROADMAP](../ROADMAP.md) and ADRs 0011–0013 describe the current code and defaults.
The designs below preserve the original planning; where there is a difference,
consult the implementation ADRs and the [deploy checklist](deploy-checklist.md).

**Planning history:** This documentation does not represent features already implemented. The requirements expressed by the user are confirmed; recommendations and proposed ADRs need answers from the grilling. No authorization to activate real operations is implied by this planning.

## Navigation

| Document | Purpose |
| --- | --- |
| [Product](product.md) | Objective, scope and traceable requirements |
| [Brainstorming](brainstorming.md) | Alternatives, tradeoffs and hypotheses |
| [Grilling](grilling.md) | Decision tree and question rounds |
| [Architecture](architecture.md) | Services, event flow and boundaries |
| [Data model](data-model.md) | Entities, integrity and precision |
| [Planning](planning.md) | Phases, dependencies and completion criteria |
| [Railway operations](railway.md) | Deploy, variables, recovery and observability |
| [Validation](validation.md) | Tests, quantitative experiments and release |
| [Sources](sources.md) | Primary documentation and limits of the verifications |
| [Deploy/configurations](deploy-checklist.md) | Variables, publication and recovery |
| [Web research](web-research.md) | Findings, official sources, limitations and impacts on the project |
| [SDDs](sdd/README.md) | Behavior specifications and technical design |
| [ADRs](adrs/README.md) | Architectural decisions and their states |

## Scope confirmed by the owner

Personal use, Brazil/B3 and US markets, backtesting and paper trading in the MVP.
Alpaca Paper in the US, the internal B3 simulator and an authorized Google account
are confirmed. Data sources, instruments, frequency, limits and identity recovery
are still being refined. Round 3 in `grilling.md`.

## How to read the states

- **Confirmed:** requirement explicitly requested in this conversation.
- **Proposed:** recommended alternative, not yet accepted by the user.
- **Pending:** depends on an answer or an investigation.
- **Implemented:** evidence in the code, distinct from the state of a decision.

## Real inventory of the codebase

Repository: `AlexGalhardo/vukem-trading`. Inspected codebase: commit `9e21d88`.
Vukem 0.41.0; Go 1.27.4; Inertia React 3.9.1; React 19.3.1;
Tailwind 4.3.3; PostgreSQL with Vukem driver 0.6.1; Vite and TypeScript.
The inventory above identifies the original codebase. The evolution implemented Google-only,
shadcn/ui sidebar, dashboard, data, strategies, backtest, paper adapters and worker.
`/posts` is no longer published. Detailed state and tests in the ROADMAP.

## Process used

The `grilling` skill was found at `/tmp/vukem/.claude/skills/grilling/SKILL.md`.
It guides interviews in rounds, with explicit dependencies between decisions.
No installed skills named `planejamento`, `brainstorming` or `grill-me` were found;
the documents with these themes are planning work, without claiming execution of
missing skills. The copy in `/tmp` is not part of the project and may disappear; this
record preserves its provenance.

The owner authorized documenting now. The product implementation does not start
at this stage. After each answer, update `grilling.md`, requirements, affected
SDDs and ADRs. Never silently promote a recommendation to a decision.

## Operations and extensions

- [Runbooks](runbooks/README.md): incident response, restore and key rotation.
- [Read-only MCP endpoint](mcp.md) and the [dependency policy](dependency-policy.md).
- [Trading knowledge base](trading/README.md) and the [setup guides](setup-local-with-docker.md).
