# RPC Boundaries

Go services use go-zero zrpc for internal calls. `gateway-api` is an HTTP gateway only and must not directly access business tables.

## strategy-rpc

Proto: `backend-go/apps/strategy-rpc/strategy.proto`

Main RPC methods:

- `ListStrategies`
- `GetStrategy`
- `CreateStrategy`
- `UpdateStrategy`
- `RunStrategy`
- `GetSignals`
- `GetSignalsBySymbol`
- `GetLatestSignal`

`strategy-rpc` calls Python through:

```text
GET /api/strategy/signal/{symbol}?strategy={python_strategy_code}
```

It persists:

- `strategies`
- `strategy_runs`
- `strategy_signals`

Events:

- `strategy.run.created`
- `strategy.run.completed`
- `strategy.run.failed`
- `strategy.signal.generated`

## backtest-rpc

Proto: `backend-go/apps/backtest-rpc/backtest.proto`

Main RPC methods:

- `RunBacktest`
- `ListBacktestTasks`
- `GetBacktestTask`
- `GetBacktestResult`
- `GetBacktestTrades`
- `GetBacktestEquityCurve`

`backtest-rpc` calls Python through:

```text
POST /api/backtest/run
```

It persists:

- `backtest_tasks`
- `backtest_results`
- `backtest_trades`
- `backtest_equity_curve`

Events:

- `backtest.task.created`
- `backtest.task.completed`
- `backtest.task.failed`

## Agent Integration

## risk-rpc

Proto: `backend-go/apps/risk-rpc/risk.proto`

Main RPC methods:

- `AnalyzeStockRisk`
- `AnalyzeAccountRisk`
- `AnalyzePositionRisk`
- `AnalyzeBacktestRisk`
- `ListRiskRecords`
- `GetRiskRecord`
- `ResolveRiskRecord`
- `IgnoreRiskRecord`

`risk-rpc` calls other Go RPC services only:

- `market-rpc` for realtime quote and full indicators.
- `strategy-rpc` for latest strategy signal.
- `paper-rpc` for paper assets, positions and equity curve.
- `backtest-rpc` for backtest task, result and equity curve.

It persists:

- `risk_records`
- `risk_rules`

Events:

- `risk.warning.created`

## Agent Integration

`agent-rpc` uses `strategyclient.Strategy`, `backtestclient.Backtest` and `riskclient.Risk` as real tools.
Temporary RPC failures degrade to tool-level warning output instead of aborting the whole agent report.

## task-rpc

Proto: `backend-go/apps/task-rpc/task.proto`

Main RPC methods:

- `CreateTask`
- `ListTasks`
- `GetTask`
- `UpdateTaskStatus`
- `CancelTask`
- `AddTaskLog`
- `ListTaskLogs`

It persists:

- `system_tasks`
- `task_logs`

It consumes `quant.task.events` and maps business events into task status.

## notify-rpc

Proto: `backend-go/apps/notify-rpc/notify.proto`

Main RPC methods:

- `CreateNotification`
- `ListNotifications`
- `CountUnread`
- `MarkRead`
- `MarkAllRead`
- `DeleteNotification`

It persists:

- `notifications`
- `notification_events`

It consumes `quant.notify.events` idempotently and publishes `notify.created`.
