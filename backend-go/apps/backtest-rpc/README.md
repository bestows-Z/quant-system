# backtest-rpc

`backtest-rpc` owns backtest task creation, Python backtest execution and result persistence.
It processes real historical market data and does not connect to a brokerage account.

## Responsibilities

- Create and run backtest tasks.
- Call `POST /api/backtest/run` on `quant-python`.
- Persist task status, execution configuration, data quality, summary result, orders, trades and equity curve atomically.
- Return saved backtest data through zrpc.
- Publish RabbitMQ events with graceful log-only fallback when RabbitMQ is unavailable.

## Python Request Mapping

Gateway/RPC strategy codes are mapped before calling Python:

- `moving_average_cross` -> `ma_cross`
- `rsi_overbought_oversold` -> `rsi_reversal`
- other strategy codes are passed through.

The Python request also carries `adjust=qfq`, commission minimum, stamp duty, transfer fee, slippage, volume participation limit, target position, lot size and tick size. Results must declare `backtest-v2` and `NEXT_OPEN_V2`; legacy or unverifiable results are rejected.
Strategy parameters are checked against a shared numeric whitelist before a task is created. Unknown names and invalid periods are rejected rather than silently replaced by engine defaults.

`GET /api/backtest/{id}/orders` exposes filled, partial, rejected and cancelled orders with dates and reasons. Existing MySQL volumes require migration `015_backtest_integrity_v2.sql` before starting this service version.

## Events

- `backtest.task.created`
- `backtest.task.completed`
- `backtest.task.failed`

## Run

```bash
go run ./apps/backtest-rpc/backtest.go -f ./apps/backtest-rpc/etc/backtest-rpc.yaml
```

All public HTTP access goes through `gateway-api` with JWT authentication and unified `{code,message,data,trace_id}` responses.
