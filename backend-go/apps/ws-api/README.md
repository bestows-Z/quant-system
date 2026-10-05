# ws-api

`ws-api` provides WebSocket realtime push.

## Paths

- `/ws/market`
- `/ws/watchlist`
- `/ws/signal`
- `/ws/paper`
- `/ws/risk`
- `/ws/agent`
- `/ws/notify`

JWT token can be provided by `Authorization: Bearer <token>` or `?token=<token>`.

## Client Command

```json
{"type":"subscribe","channel":"risk","symbol":"000001"}
```

```json
{"type":"unsubscribe","channel":"risk","symbol":"000001"}
```

## Server Message

```json
{
  "type": "risk.warning",
  "channel": "risk",
  "trace_id": "trace",
  "event_id": "event",
  "data": {},
  "created_at": "2026-07-03T00:00:00Z"
}
```

RabbitMQ downtime degrades to logs and does not block WebSocket startup.
