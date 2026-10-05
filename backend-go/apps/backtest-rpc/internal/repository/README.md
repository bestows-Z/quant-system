# repository

Repository boundary for backtest persistence.

The first runnable version keeps SQL orchestration in `internal/domain` to preserve transactional clarity.
Future rounds can move task, result, trade and equity SQL into this package without changing RPC contracts.
