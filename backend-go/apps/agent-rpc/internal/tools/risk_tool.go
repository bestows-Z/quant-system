package tools

import (
	"context"
	"fmt"
	"time"

	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/risk-rpc/riskclient"
)

type RiskTool struct{ Risk riskclient.Risk }

func (t RiskTool) Name() string { return "risk_tool" }
func (t RiskTool) Description() string {
	return "调用 risk-rpc 生成个股、账户、持仓和回测风险分析"
}

func (t RiskTool) Execute(ctx context.Context, input domain.ToolInput) (domain.ToolOutput, error) {
	start := time.Now()
	if t.Risk == nil {
		data := map[string]any{"degraded": true, "message": "risk-rpc client 未配置，已降级"}
		return domain.ToolOutput{ToolName: t.Name(), Data: data, Error: "risk-rpc client nil", Success: false, LatencyMs: time.Since(start).Milliseconds()}, nil
	}
	data := map[string]any{}
	if input.Symbol != "" {
		reply, err := t.Risk.AnalyzeStockRisk(ctx, &riskclient.AnalyzeStockRiskReq{UserId: input.UserID, Symbol: input.Symbol})
		if err != nil {
			return t.degraded(start, err, data), nil
		}
		data["stock_risk"] = reply
	}
	if input.AccountID > 0 {
		account, err := t.Risk.AnalyzeAccountRisk(ctx, &riskclient.AnalyzeAccountRiskReq{UserId: input.UserID, AccountId: input.AccountID})
		if err != nil {
			return t.degraded(start, err, data), nil
		}
		data["account_risk"] = account
		position, err := t.Risk.AnalyzePositionRisk(ctx, &riskclient.AnalyzePositionRiskReq{UserId: input.UserID, AccountId: input.AccountID, Symbol: input.Symbol})
		if err != nil {
			return t.degraded(start, err, data), nil
		}
		data["position_risk"] = position
	}
	if id := extraUint(input.Extra, "backtest_id"); id > 0 {
		reply, err := t.Risk.AnalyzeBacktestRisk(ctx, &riskclient.AnalyzeBacktestRiskReq{UserId: input.UserID, TaskId: id})
		if err != nil {
			return t.degraded(start, err, data), nil
		}
		data["backtest_risk"] = reply
	}
	if len(data) == 0 {
		data["message"] = "未提供 symbol、account_id 或 backtest_id，risk-rpc 未执行具体分析"
	}
	return domain.ToolOutput{ToolName: t.Name(), Data: data, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
}

func (t RiskTool) degraded(start time.Time, err error, partial map[string]any) domain.ToolOutput {
	partial["degraded"] = true
	partial["error"] = err.Error()
	partial["message"] = "risk-rpc 调用失败，已降级"
	return domain.ToolOutput{ToolName: t.Name(), Data: partial, Error: err.Error(), Success: false, LatencyMs: time.Since(start).Milliseconds()}
}

func extraUint(extra map[string]any, key string) uint64 {
	switch v := extra[key].(type) {
	case uint64:
		return v
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	case float64:
		return uint64(v)
	case string:
		var out uint64
		_, _ = fmt.Sscan(v, &out)
		return out
	default:
		return 0
	}
}
