# agent-rpc

`agent-rpc` is the quant agent orchestration service. It is not a broker or trading executor. It combines market, indicator, RAG, paper-account, strategy, risk and backtest context into structured research reports.

## Implemented

- go-zero RPC service on `0.0.0.0:9009`
- `AnalyzeStock`, `Ask`, `ExplainSignal`, `ReviewBacktest`, `ReviewPaperAccount`
- Task list, task detail and report detail
- Real tools:
  - `MarketTool` calls `market-rpc`
  - `IndicatorTool` calls `market-rpc`
  - `StrategyTool` calls `strategy-rpc`
  - `RiskTool` calls `risk-rpc`
  - `BacktestTool` calls `backtest-rpc`
  - `RagSearchTool` calls `rag-rpc`
  - `PaperAccountTool` calls `paper-rpc`
- Always-available local rules engine that synthesizes structured reports from real tool results
- Real OpenAI-compatible HTTP client enabled through `LLM_PROVIDER`, `LLM_BASE_URL`, `LLM_API_KEY` and `LLM_MODEL`
- Task, message, tool-call and report persistence
- RabbitMQ events:
  - `agent.task.created`
  - `agent.task.completed`
  - `agent.task.failed`

## Run

```bash
cd backend-go
go run ./apps/agent-rpc/agent.go -f ./apps/agent-rpc/etc/agent-rpc.yaml
```

## Verify

```bash
cd backend-go
GOCACHE=$PWD/.gocache go mod tidy
GOCACHE=$PWD/.gocache go test ./...
GOCACHE=$PWD/.gocache go build ./...
```

All outputs include `risk_level`, `confidence` and `disclaimer`. The service only performs data analysis and simulation research; it never creates real trading actions.
