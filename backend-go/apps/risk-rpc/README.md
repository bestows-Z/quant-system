# risk-rpc

`risk-rpc` is the risk-control service for research analysis and paper trading.
It does not connect to brokers and does not produce deterministic buy/sell advice.

## Responsibilities

- Analyze stock risk through `market-rpc` and `strategy-rpc`.
- Analyze paper account and position risk through `paper-rpc`.
- Analyze backtest risk through `backtest-rpc`.
- Persist risk records in `risk_records`.
- Seed system risk rules in `risk_rules`.
- Publish `risk.warning.created` when a HIGH or EXTREME item is generated.

## Risk Levels

- `LOW`: 0-30
- `MEDIUM`: 31-60
- `HIGH`: 61-80
- `EXTREME`: 81-100

## Run

```bash
cd backend-go
go run ./apps/risk-rpc/risk.go -f ./apps/risk-rpc/etc/risk-rpc.yaml
```

All public HTTP access goes through `gateway-api` with JWT authentication and unified `{code,message,data,trace_id}` responses.
