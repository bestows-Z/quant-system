# Agent Service

`agent-rpc` orchestrates quant research workflows. It calls `market-rpc`, `strategy-rpc`, `backtest-rpc`, `risk-rpc`, `rag-rpc` and `paper-rpc` through zrpc, stores every task/message/tool call/report in MySQL, and returns structured reports through `gateway-api`.

First-version real calls:

- Market and realtime context from `market-rpc`
- Indicator context from `market-rpc`
- Strategy signal context from `strategy-rpc`
- Backtest review context from `backtest-rpc`
- Risk analysis context from `risk-rpc`
- Knowledge context from `rag-rpc`
- Paper account context from `paper-rpc`

First-version mocks:

- LLM provider

If strategy-rpc, backtest-rpc or risk-rpc is unavailable, Agent returns a degraded tool result and keeps the research task alive.

The agent does not connect to brokers and does not place real orders. All reports are research-only and do not constitute investment advice.
