# Event Bus

RabbitMQ is the preferred event bus.

Unified event:

```json
{
  "event_id": "",
  "event_type": "",
  "trace_id": "",
  "user_id": 0,
  "payload": {},
  "created_at": ""
}
```

Event types:
- `market.quote.updated`
- `strategy.signal.generated`
- `strategy.run.created`
- `strategy.run.completed`
- `strategy.run.failed`
- `backtest.task.created`
- `backtest.task.completed`
- `backtest.task.failed`
- `paper.order.created`
- `paper.order.filled`
- `paper.order.canceled`
- `risk.warning.created`
- `rag.document.uploaded`
- `rag.document.chunked`
- `rag.embedding.completed`
- `task.created`
- `task.updated`
- `task.completed`
- `task.failed`
- `agent.task.created`
- `agent.task.completed`
- `agent.task.failed`
- `notify.created`
- `ws.message.pushed`

Current round status:
- Producer wrapper exists in `common/xmq`.
- Producer declares topic exchange and publishes by event type.
- If RabbitMQ is unavailable, producer degrades to logging and returns nil, so core HTTP/RPC flows can continue.
- strategy-rpc publishes strategy run and signal events.
- backtest-rpc publishes task lifecycle events.
- risk-rpc publishes warning events for HIGH and EXTREME risk items.
- task-rpc consumes `quant.task.events` and synchronizes `system_tasks`/`task_logs`.
- notify-rpc consumes `quant.notify.events` and writes `notifications`/`notification_events`.
- ws-api consumes `quant.ws.events` and pushes messages to connected WebSocket clients.
- Consumer implementation now exists in `common/xmq` and is used by task-rpc, notify-rpc and ws-api. Auto reconnect and dead-letter queues remain future enhancements.
