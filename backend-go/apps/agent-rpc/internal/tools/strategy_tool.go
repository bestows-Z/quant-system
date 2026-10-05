package tools

import (
	"context"
	"time"

	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"
)

type StrategyTool struct{ Strategy strategyclient.Strategy }

func (t StrategyTool) Name() string { return "strategy_tool" }
func (t StrategyTool) Description() string {
	return "调用 strategy-rpc 获取策略信号"
}

func (t StrategyTool) Execute(ctx context.Context, input domain.ToolInput) (domain.ToolOutput, error) {
	start := time.Now()
	if t.Strategy == nil {
		data := map[string]any{"symbol": input.Symbol, "signal": "HOLD", "score": 0.55, "reason": "strategy-rpc client 未配置，降级为 mock", "risk_level": domain.RiskMedium}
		return domain.ToolOutput{ToolName: t.Name(), Data: data, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
	}
	reply, err := t.Strategy.RunStrategy(ctx, &strategyclient.RunStrategyReq{UserId: input.UserID, StrategyCode: "trend_score", Symbol: input.Symbol, ParamsJson: "{}"})
	if err != nil {
		data := map[string]any{"degraded": true, "error": err.Error(), "symbol": input.Symbol, "signal": "HOLD", "reason": "strategy-rpc 调用失败，已降级"}
		return domain.ToolOutput{ToolName: t.Name(), Data: data, Error: err.Error(), Success: false, LatencyMs: time.Since(start).Milliseconds()}, nil
	}
	data := map[string]any{"run_id": reply.RunId, "signals": reply.Signals, "summary": reply.Summary, "disclaimer": reply.Disclaimer}
	return domain.ToolOutput{ToolName: t.Name(), Data: data, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
}
