# Gateway API Plan

The gateway will expose:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/user/profile`
- `GET /api/stocks`
- `GET /api/stocks/search`
- `GET /api/stocks/:symbol`
- `GET /api/market/realtime`
- `GET /api/market/realtime/:symbol`
- `GET /api/kline/:symbol`
- `GET /api/indicator/:symbol`
- `GET /api/indicator/full/:symbol`
- `GET /api/watchlists`
- `POST /api/watchlists`
- `DELETE /api/watchlists/:id`
- `PUT /api/watchlists/:id`
- `GET /api/strategies`
- `GET /api/strategies/:id`
- `POST /api/strategies`
- `PUT /api/strategies/:id`
- `POST /api/strategies/:id/run`
- `GET /api/signals`
- `GET /api/signals/:symbol`
- `GET /api/signals/:symbol/latest`
- `POST /api/backtest/run`
- `GET /api/backtest/tasks`
- `GET /api/backtest/:id`
- `GET /api/backtest/:id/result`
- `GET /api/backtest/:id/trades`
- `GET /api/backtest/:id/equity-curve`
- `GET /api/risk/records`
- `GET /api/risk/records/:id`
- `POST /api/risk/stock/:symbol/analyze`
- `POST /api/risk/account/:accountId/analyze`
- `POST /api/risk/account/:accountId/positions/analyze`
- `POST /api/risk/backtest/:taskId/analyze`
- `PUT /api/risk/records/:id/resolve`
- `PUT /api/risk/records/:id/ignore`
- `POST /api/tasks`
- `GET /api/tasks`
- `GET /api/tasks/:id`
- `PUT /api/tasks/:id/status`
- `POST /api/tasks/:id/cancel`
- `POST /api/tasks/:id/logs`
- `GET /api/tasks/:id/logs`
- `GET /api/notifications`
- `GET /api/notifications/unread-count`
- `PUT /api/notifications/:id/read`
- `PUT /api/notifications/read-all`
- `DELETE /api/notifications/:id`
- `POST /api/paper/accounts`
- `GET /api/paper/accounts`
- `GET /api/paper/accounts/:id`
- `GET /api/paper/accounts/:id/assets`
- `GET /api/paper/accounts/:id/positions`
- `POST /api/paper/orders`
- `DELETE /api/paper/orders/:id`
- `GET /api/paper/orders`
- `GET /api/paper/orders/:id`
- `GET /api/paper/trades`
- `GET /api/paper/equity-curve`
- `POST /api/rag/knowledge-bases`
- `GET /api/rag/knowledge-bases`
- `GET /api/rag/knowledge-bases/:id`
- `POST /api/rag/documents`
- `GET /api/rag/documents`
- `GET /api/rag/documents/:id`
- `GET /api/rag/chunks`
- `POST /api/rag/search`
- `POST /api/rag/ask`
- `POST /api/agent/analyze-stock`
- `POST /api/agent/explain-signal`
- `POST /api/agent/review-backtest`
- `POST /api/agent/review-paper-account`
- `POST /api/agent/ask`
- `GET /api/agent/tasks`
- `GET /api/agent/tasks/:id`
- `GET /api/agent/reports/:id`
- `GET /api/system/health`

The gateway must call RPC services through zrpc and must not directly access business tables.

## Round 3 Implemented

Market gateway routes now call `market-rpc` through zrpc:

- `GET /api/stocks`
- `GET /api/stocks/search?keyword=`
- `GET /api/stocks/:symbol`
- `GET /api/market/realtime`
- `GET /api/market/realtime/:symbol`
- `GET /api/kline/:symbol?period=daily&start=&end=`
- `GET /api/indicator/:symbol?type=macd&start=&end=`
- `GET /api/indicator/full/:symbol?start=&end=`
- `GET /api/watchlists`
- `POST /api/watchlists`
- `DELETE /api/watchlists/:id`

`market-rpc` calls the existing `quant-python` HTTP API through a dedicated client under `apps/market-rpc/internal/client`.
Realtime quote responses are cached in Redis and `market.quote.updated` events are published through RabbitMQ. If RabbitMQ is unavailable, `common/xmq` degrades to logging.

## Round 4 Implemented

Paper trading gateway routes are JWT protected and call `paper-rpc` through zrpc. The gateway does not access MySQL directly.

- `POST /api/paper/accounts`
  - body: `{"account_name":"研究账户","initial_cash":"100000"}`
- `GET /api/paper/accounts`
- `GET /api/paper/accounts/:id`
- `GET /api/paper/accounts/:id/assets`
- `GET /api/paper/accounts/:id/positions`
- `POST /api/paper/orders`
  - body: `{"account_id":1,"client_order_id":"client-001","symbol":"000001","side":"BUY","order_type":"MARKET","price":"","volume":100}`
- `DELETE /api/paper/orders/:id`
- `GET /api/paper/orders?account_id=&symbol=&status=`
- `GET /api/paper/orders/:id`
- `GET /api/paper/trades?account_id=&symbol=`
- `GET /api/paper/equity-curve?account_id=`

`paper-rpc` uses `market-rpc` for the latest quote, wraps fund/order/trade/position/equity updates in MySQL transactions, and publishes `paper.order.created`, `paper.order.filled`, and `paper.order.canceled` events through RabbitMQ. RabbitMQ downtime degrades to structured logs.

## Round 5 Implemented

RAG gateway routes are JWT protected and call `rag-rpc` through zrpc. The first version uses deterministic mock embeddings, MySQL metadata/chunk storage, and a reserved Qdrant client boundary.

- `POST /api/rag/knowledge-bases`
  - body: `{"name":"量化研究资料","description":"策略和系统文档","visibility":"PRIVATE"}`
- `GET /api/rag/knowledge-bases`
- `GET /api/rag/knowledge-bases/:id`
- `POST /api/rag/documents`
  - body: `{"knowledge_base_id":1,"title":"MA策略说明","doc_type":"text","content":"...","metadata_json":"{}"}`
- `GET /api/rag/documents?knowledge_base_id=1`
- `GET /api/rag/documents/:id`
- `GET /api/rag/chunks?document_id=1`
- `POST /api/rag/search`
  - body: `{"knowledge_base_id":1,"query":"均线策略如何判断趋势","top_k":5}`
- `POST /api/rag/ask`
  - body: `{"knowledge_base_id":1,"question":"均线策略的风险是什么","top_k":5}`

`rag-rpc` immediately chunks uploaded text, writes mock embeddings into `rag_embeddings.vector_json`, logs retrieval/QA calls, and publishes `rag.document.uploaded`, `rag.document.chunked`, and `rag.embedding.completed` events. The generated answer is a mock answer based on retrieved chunks and is explicitly marked as research-only.

## Round 6 Implemented

Agent gateway routes are JWT protected and call `agent-rpc` through zrpc. The gateway never trusts a client-provided `user_id`; it injects the authenticated user from JWT.

- `POST /api/agent/analyze-stock`
  - body: `{"symbol":"000001","question":"结合行情、指标和知识库分析当前状态","knowledge_base_id":1,"account_id":1}`
- `POST /api/agent/explain-signal`
  - body: `{"symbol":"000001","question":"解释当前策略信号","knowledge_base_id":1}`
- `POST /api/agent/review-backtest`
  - body: `{"backtest_id":1,"question":"复盘这次回测","knowledge_base_id":1}`
- `POST /api/agent/review-paper-account`
  - body: `{"account_id":1,"question":"分析模拟账户风险","knowledge_base_id":1}`
- `POST /api/agent/ask`
  - body: `{"question":"结合知识库回答一个量化研究问题","symbol":"000001","knowledge_base_id":1,"account_id":1}`
- `GET /api/agent/tasks?status=&task_type=&limit=`
- `GET /api/agent/tasks/:id`
- `GET /api/agent/reports/:id`

`agent-rpc` persists tasks, messages, tool calls and reports. It calls real `market-rpc`, `rag-rpc`, `paper-rpc`, `strategy-rpc`, `backtest-rpc` and `risk-rpc` tools. Events `agent.task.created`, `agent.task.completed` and `agent.task.failed` are published through RabbitMQ with degraded logging when RabbitMQ is unavailable.

## Round 7 Implemented

Strategy and backtest gateway routes are JWT protected and call dedicated go-zero RPC services through zrpc. The gateway does not call `quant-python` or MySQL for these domains.

Strategy routes:

- `GET /api/strategies`
- `GET /api/strategies/:id`
- `POST /api/strategies`
  - body: `{"code":"trend_score_custom","name":"趋势评分扩展","description":"仅供研究","params_json":"{}"}`
- `PUT /api/strategies/:id`
- `POST /api/strategies/:id/run`
  - body: `{"symbol":"000001","start_date":"2024-01-01","end_date":"2024-12-31","params_json":"{}"}`
- `GET /api/signals?symbol=&strategy_code=&limit=`
- `GET /api/signals/:symbol`
- `GET /api/signals/:symbol/latest`

`strategy-rpc` calls `quant-python` through `GET /api/strategy/signal/{symbol}?strategy=...`, persists `strategy_runs` and `strategy_signals`, and publishes `strategy.run.created`, `strategy.run.completed`, `strategy.run.failed` and `strategy.signal.generated`.

Backtest routes:

- `POST /api/backtest/run`
  - body: `{"symbol":"000001","strategy_code":"moving_average_cross","start_date":"2024-01-01","end_date":"2024-12-31","initial_cash":"100000","fee_rate":"0.0003","slippage_rate":"0.0001","params_json":"{}"}`
- `GET /api/backtest/tasks?symbol=&status=&limit=`
- `GET /api/backtest/:id`
- `GET /api/backtest/:id/result`
- `GET /api/backtest/:id/trades`
- `GET /api/backtest/:id/equity-curve`

`backtest-rpc` calls `quant-python` through `POST /api/backtest/run`, persists `backtest_tasks`, `backtest_results`, `backtest_trades` and `backtest_equity_curve`, and publishes `backtest.task.created`, `backtest.task.completed` and `backtest.task.failed`.

## Round 8 Implemented

Risk gateway routes are JWT protected and call `risk-rpc` through zrpc. The gateway does not access risk tables directly.

- `GET /api/risk/records?account_id=&symbol=&risk_type=&risk_level=&status=&limit=`
- `GET /api/risk/records/:id`
- `POST /api/risk/stock/:symbol/analyze`
- `POST /api/risk/account/:id/analyze`
- `POST /api/risk/account/:id/positions/analyze?symbol=`
- `POST /api/risk/backtest/:id/analyze`
- `PUT /api/risk/records/:id/resolve`
- `PUT /api/risk/records/:id/ignore`

`risk-rpc` analyzes:

- stock risk by calling `market-rpc` realtime quote/full indicators and `strategy-rpc` latest signal;
- account and position risk by calling `paper-rpc` assets, positions and equity curve;
- backtest risk by calling `backtest-rpc` task/result/equity curve.

Each analysis writes risk items into `risk_records`. HIGH and EXTREME items publish `risk.warning.created`; RabbitMQ downtime degrades to logs through `common/xmq`.

## Round 9 Implemented

Task and notification gateway routes are JWT protected and call dedicated go-zero RPC services through zrpc.

Task routes:

- `POST /api/tasks`
- `GET /api/tasks?task_type=&status=&source_type=&source_id=&limit=`
- `GET /api/tasks/:id`
- `PUT /api/tasks/:id/status`
- `POST /api/tasks/:id/cancel`
- `POST /api/tasks/:id/logs`
- `GET /api/tasks/:id/logs`

Notification routes:

- `GET /api/notifications?notify_type=&level=&unread_only=&limit=`
- `GET /api/notifications/unread-count`
- `PUT /api/notifications/:id/read`
- `PUT /api/notifications/read-all`
- `DELETE /api/notifications/:id`

`ws-api` exposes `/ws/market`, `/ws/watchlist`, `/ws/signal`, `/ws/paper`, `/ws/risk`, `/ws/agent` and `/ws/notify`. JWT is accepted through `Authorization: Bearer <token>` or `?token=`.
