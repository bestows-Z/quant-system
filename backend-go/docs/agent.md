# Agent Design

`agent-rpc` is implemented as the quant analysis orchestration service. It does not perform real trading and never places orders automatically.

## Workflow

`AnalyzeStock` creates an `agent_task`, stores the user message, marks the task as running, executes tools, builds context and prompt, calls the LLM provider, writes an `agent_analysis_report`, stores the assistant message, and publishes an event.

## Tool Interface

```go
type AgentTool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, input ToolInput) (ToolOutput, error)
}
```

Real tools:

- `MarketTool`: calls `market-rpc` for stock detail, realtime quote and K-line data.
- `IndicatorTool`: calls `market-rpc` for indicators.
- `StrategyTool`: calls `strategy-rpc` to generate and persist research signals.
- `BacktestTool`: calls `backtest-rpc` to run and persist a simple research backtest.
- `RiskTool`: calls `risk-rpc` for stock, account, position and backtest risk.
- `RagSearchTool`: calls `rag-rpc` search.
- `PaperAccountTool`: calls `paper-rpc` for assets, positions, orders, trades and equity curve.

Mock tools with explicit replacement boundaries:

- none for data tools in the current round

If strategy-rpc, backtest-rpc or risk-rpc is temporarily unavailable, the corresponding tool returns a degraded tool output instead of failing the entire agent task.

## Persistence

- `agent_tasks`
- `agent_messages`
- `agent_tool_calls`
- `agent_analysis_reports`

## Events

- `agent.task.created`
- `agent.task.completed`
- `agent.task.failed`

RabbitMQ downtime degrades to logging through `common/xmq`.

## Output

Every report includes `summary`, `market_analysis`, `technical_analysis`, `strategy_analysis`, `rag_analysis`, `paper_trading_analysis`, `risk_analysis`, `final_view`, `confidence`, `risk_level`, `references` and `disclaimer`.

The disclaimer is always `仅供研究，不构成投资建议`.
