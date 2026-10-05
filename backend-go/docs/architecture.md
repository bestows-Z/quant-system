# Go Backend Architecture

后端采用 go-zero API Gateway + zrpc 微服务架构。

```text
gateway-api
  | zrpc
  +-- user-rpc
  +-- market-rpc ---- HTTP ---- quant-python
  +-- strategy-rpc
  +-- backtest-rpc ---- HTTP ---- quant-python
  +-- paper-rpc
  +-- risk-rpc
  +-- rag-rpc ---- HTTP ---- qdrant
  +-- agent-rpc ---- zrpc ---- market/strategy/backtest/risk/rag/paper
  +-- task-rpc
  +-- notify-rpc

ws-api consumes events and pushes realtime messages.
RabbitMQ is the event bus.
etcd is service registry and discovery.
```

Rules:
- `gateway-api` exposes HTTP APIs and Web entry contracts, but does not own core business logic.
- `gateway-api` must call business services through zrpc.
- RPC services own business rules, persistence, cache, events and integration with Python or external services.
- `quant-python` is only accessed by Go services, never directly by the frontend.
- Agent and RAG can use mock providers in v1, but provider interfaces must stay replaceable.
- All analysis and paper trading responses must include “仅供研究，不构成投资建议”.

Current round status:
- user, market, strategy, backtest, risk, paper, rag and agent RPC services have runnable first-version implementations.
- task-rpc, notify-rpc and ws-api provide the first event-driven task/notification/realtime push loop.
- strategy-rpc calls `quant-python` `GET /api/strategy/signal/{symbol}` and persists runs/signals.
- backtest-rpc calls `quant-python` `POST /api/backtest/run` and persists task, result, trades and equity curve.
- agent-rpc now uses real strategy, backtest and risk RPC tools.
- Real common Go packages exist for response, errors, JWT, middleware, xhttp, xmq, xqdrant, xllm and decimal.
