# paper-rpc

`paper-rpc` implements the paper trading core. It is a research-only virtual matching service and does not connect to any broker or real trading channel.

## Capabilities

- Create and list paper accounts
- Query assets, positions, orders, trades and equity curve
- Submit market and limit orders
- Cancel pending limit orders
- Use `client_order_id` for idempotent order submission
- Use MySQL transactions for account cash, frozen cash, positions, orders, trades and equity curve
- Fetch latest quote through `market-rpc`
- Publish RabbitMQ events:
  - `paper.order.created`
  - `paper.order.filled`
  - `paper.order.canceled`

## Run

```bash
cd backend-go
go run ./apps/paper-rpc/paper.go -f ./apps/paper-rpc/etc/paper-rpc.yaml
```

Required dependencies:

- MySQL initialized with `deploy/mysql/init.sql`
- etcd
- market-rpc
- RabbitMQ, optional at runtime; unavailable RabbitMQ degrades to logging

## Verify

```bash
cd backend-go
GOCACHE=$PWD/.gocache go test ./...
GOCACHE=$PWD/.gocache go build ./...
```

All API responses are exposed through `gateway-api` with JWT authentication. This service returns RPC objects to the gateway; the gateway wraps them in `{code,message,data,trace_id}`.
