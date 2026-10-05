# WebSocket

`ws-api` exposes realtime channels for the quant platform.

## Connection

Use JWT:

```text
ws://localhost:8081/ws/notify?token=<jwt>
```

or:

```text
Authorization: Bearer <jwt>
```

## Channels

- `market`
- `watchlist`
- `signal`
- `paper`
- `risk`
- `agent`
- `notify`

## Subscribe

```json
{"type":"subscribe","channel":"risk","symbol":"000001"}
```

```json
{"type":"subscribe","channel":"paper","account_id":1}
```

## Message

```json
{
  "type": "risk.warning",
  "channel": "risk",
  "trace_id": "xxx",
  "event_id": "xxx",
  "data": {},
  "created_at": "2026-07-03T00:00:00Z"
}
```

## RabbitMQ Sources

- `market.quote.updated`
- `strategy.signal.generated`
- `paper.order.filled`
- `risk.warning.created`
- `agent.task.completed`
- `agent.task.failed`
- `notify.created`
