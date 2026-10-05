package tools

import (
	"context"
	"time"

	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
)

type BacktestTool struct{ Backtest backtestclient.Backtest }

func (t BacktestTool) Name() string { return "backtest_tool" }
func (t BacktestTool) Description() string {
	return "调用 backtest-rpc 获取或生成回测结果"
}

func (t BacktestTool) Execute(ctx context.Context, input domain.ToolInput) (domain.ToolOutput, error) {
	start := time.Now()
	if t.Backtest == nil || input.Symbol == "" {
		data := map[string]any{"symbol": input.Symbol, "available": false, "message": "backtest-rpc client 未配置或无 symbol，跳过"}
		return domain.ToolOutput{ToolName: t.Name(), Data: data, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
	}
	reply, err := t.Backtest.RunBacktest(ctx, &backtestclient.RunBacktestReq{UserId: input.UserID, Symbol: input.Symbol, StrategyCode: "trend_score", StartDate: "2024-01-01", EndDate: "2024-12-31", InitialCash: "100000", FeeRate: "0.0003", SlippageRate: "0.0005", ParamsJson: "{}"})
	if err != nil {
		data := map[string]any{"degraded": true, "error": err.Error(), "symbol": input.Symbol, "message": "backtest-rpc 调用失败，已降级"}
		return domain.ToolOutput{ToolName: t.Name(), Data: data, Error: err.Error(), Success: false, LatencyMs: time.Since(start).Milliseconds()}, nil
	}
	data := map[string]any{"task_id": reply.TaskId, "task_no": reply.TaskNo, "result": reply.Result, "trades": reply.Trades, "equity_curve": reply.EquityCurve, "disclaimer": reply.Disclaimer}
	return domain.ToolOutput{ToolName: t.Name(), Data: data, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
}
