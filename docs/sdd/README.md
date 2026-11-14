# SDDs — specifications and software design

SDD in this project means **Software Design Document**, used as a specification
for implementation and verification. Each SDD describes behavior, contracts,
failures, acceptance criteria and open items. It is not code already implemented.

| ID | Document | State |
| --- | --- | --- |
| SDD-01 | [Identity and dashboard](01-identity-dashboard.md) | Confirmed requirements; proposed details |
| SDD-02 | [Data and strategies](02-data-strategies.md) | Markets confirmed; feeds and parameters pending |
| SDD-03 | [Risk and execution](03-risk-execution.md) | Paper confirmed; policies and design proposed |
| SDD-04 | [Backtesting](04-backtesting.md) | Proposed |
| SDD-05 | [Railway and operations](05-railway-operations.md) | Destination confirmed; design proposed |
| SDD-06 | [Adapters and simulation](06-adapters-simulation.md) | Adapters confirmed; data pending |
| SDD-07 | [Daily ranking, backtest and paper runner](07-daily-ranking-and-paper-runner.md) | Implemented |

Changing a contract requires reviewing the associated ADR, data model, plan and tests.
