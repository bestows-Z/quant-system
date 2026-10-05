# repository

Repository boundary for strategy persistence.

The first runnable version keeps SQL orchestration in `internal/domain` to reduce wiring risk.
Future rounds can move strategy, run and signal SQL into this package without changing RPC contracts.
