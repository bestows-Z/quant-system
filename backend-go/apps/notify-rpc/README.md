# notify-rpc

`notify-rpc` is the notification center.

## Responsibilities

- Create notifications.
- List notifications.
- Count unread notifications.
- Mark one or all notifications as read.
- Delete notifications by soft delete.
- Consume RabbitMQ events from `quant.notify.events`.
- Store idempotent event handling records in `notification_events`.

## Event Mapping

- `strategy.signal.generated` -> strategy signal notification
- `backtest.task.completed` -> backtest success notification
- `backtest.task.failed` -> backtest failure notification
- `paper.order.filled` -> paper trade notification
- `paper.order.canceled` -> paper order cancel notification
- `risk.warning.created` -> risk warning notification
- `agent.task.completed` -> agent success notification
- `agent.task.failed` -> agent failure notification

## Run

```bash
cd backend-go
go run ./apps/notify-rpc/notify.go -f ./apps/notify-rpc/etc/notify-rpc.yaml
```
