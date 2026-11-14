# ADRs — architecture decisions

States: **accepted** when derived from an explicit requirement; **proposed** when
it needs the user's acceptance. Accepting an ADR does not mean implementing it.
Superseded ADRs remain in the history, with a link to the successor.

| ADR | Topic | State |
| --- | --- | --- |
| [0001](0001-stack-and-dashboard.md) | Stack, dashboard and shadcn/ui | Accepted by requirement; partially implemented |
| [0002](0002-modular-monolith-worker.md) | Modular monolith with separate worker | Proposed |
| [0003](0003-google-only.md) | Google-only login | Accepted by requirement; implementation pending |
| [0004](0004-postgres-outbox.md) | PostgreSQL, outbox and financial precision | Proposed; PostgreSQL already confirmed |
| [0005](0005-railway.md) | Railway deploy | Accepted by requirement; proposed topology |
| [0006](0006-data-and-events.md) | Data, extensions and messaging | Proposed |
| [0007](0007-paper-first.md) | Paper-first and deterministic execution | Paper-first accepted; details proposed |
| [0008](0008-brazil-and-us.md) | Brazil and US in the MVP | Accepted; providers pending |
| [0009](0009-personal-use.md) | Personal terminal and single account | Accepted |
| [0010](0010-paper-adapters.md) | Alpaca Paper US and B3 simulator | Implemented; external validation pending |
| [0011](0011-autonomous-implementation.md) | Defaults and autonomous implementation | Accepted by later instruction |
| [0012](0012-transactional-snapshots.md) | Persistence effectively implemented | Implemented; details/updates ADR-0004 |
| [0013](0013-polling-workers-and-deploy.md) | Polling, leadership and Railway standby | Implemented |
| [0014](0014-railway-api-and-portable-docker.md) | Current Railway API and portable Dockerfile | Implemented during publication |
| [0015](0015-bun-biome-and-bun-check.md) | Bun 1.4.4, Biome 2 and `bun check` | Implemented |
| [0016](0016-fill-rules-and-validation-protocol.md) | Fill rules, resting orders and the validation protocol | Implemented |
| [0017](0017-documentation-site.md) | Documentation site generated from `docs/` | Implemented |
| [0018](0018-data-providers-and-research-account.md) | Data-provider interface and the research account | Implemented |
| [0019](0019-workspace-ticket-and-wire-types.md) | Server-side workspace, order ticket and generated wire types | Implemented |
| [0020](0020-operations-metrics-mcp-and-drafts.md) | Operational metrics, read-only MCP and strategy drafts | Implemented |
| [0021](0021-live-readiness-gate.md) | Live-readiness gate | Proposed |
| [0022](0022-stream-testnet-adapter-and-quotes.md) | Change stream, Binance testnet adapter and quote engine | Implemented |
| [0023](0023-daily-ranking-and-paper-runner.md) | Daily ranking, end-of-day data and the paper runner | Implemented |

Each ADR must record context, decision, alternatives, consequences and review
trigger. Brazil and US and personal use are confirmed; Alpaca Paper, the internal B3 simulator and admission of one account are also. IEX and B3 CSV were adopted by delegation; credentials/datasets remain external.
