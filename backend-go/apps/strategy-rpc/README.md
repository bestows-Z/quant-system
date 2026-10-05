# strategy-rpc

`strategy-rpc` owns strategy definitions, strategy runs and generated signals.
It calls `quant-python` through `apps/strategy-rpc/internal/client` and never exposes Python HTTP details to gateway handlers.

## Responsibilities

- Manage strategy definitions: list, detail, create and update.
- Seed system strategies on startup.
- Run a strategy for one symbol.
- Call `GET /api/strategy/signal/{symbol}?strategy=...` on `quant-python`.
- Persist strategy runs and generated signals.
- Publish RabbitMQ events with graceful log-only fallback when RabbitMQ is unavailable.

## System Strategies

- `moving_average_cross` maps to Python `ma_cross`
- `macd_cross` maps to Python `macd_cross`
- `rsi_overbought_oversold` maps to Python `rsi_reversal`
- `boll_breakout` maps to Python `boll_breakout`
- `volume_breakout` maps to Python `volume_breakout`
- `trend_score` maps to Python `trend_score`

## Events

- `strategy.run.created`
- `strategy.run.completed`
- `strategy.run.failed`
- `strategy.signal.generated`

## Run

```bash
go run ./apps/strategy-rpc/strategy.go -f ./apps/strategy-rpc/etc/strategy-rpc.yaml
```

All public HTTP access goes through `gateway-api` with JWT authentication and unified `{code,message,data,trace_id}` responses.
