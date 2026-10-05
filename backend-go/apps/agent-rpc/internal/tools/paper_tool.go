package tools

import (
	"context"
	"time"

	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
)

type PaperAccountTool struct{ Paper paperclient.Paper }

func (t PaperAccountTool) Name() string { return "paper_account_tool" }
func (t PaperAccountTool) Description() string {
	return "获取模拟账户资产、持仓、订单、成交和权益曲线"
}

func (t PaperAccountTool) Execute(ctx context.Context, input domain.ToolInput) (domain.ToolOutput, error) {
	start := time.Now()
	if input.AccountID == 0 {
		return domain.ToolOutput{ToolName: t.Name(), Data: map[string]any{"message": "未提供 account_id，跳过模拟账户分析"}, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
	}
	data := map[string]any{}
	if reply, err := t.Paper.GetAccountAssets(ctx, &paperclient.GetAccountAssetsReq{UserId: input.UserID, AccountId: input.AccountID}); err == nil {
		data["assets"] = reply.Account
	}
	if reply, err := t.Paper.ListPositions(ctx, &paperclient.ListPositionsReq{UserId: input.UserID, AccountId: input.AccountID}); err == nil {
		data["positions"] = reply.Items
	}
	if reply, err := t.Paper.ListOrders(ctx, &paperclient.ListOrdersReq{UserId: input.UserID, AccountId: input.AccountID}); err == nil {
		data["orders"] = reply.Items
	}
	if reply, err := t.Paper.ListTrades(ctx, &paperclient.ListTradesReq{UserId: input.UserID, AccountId: input.AccountID}); err == nil {
		data["trades"] = reply.Items
	}
	if reply, err := t.Paper.GetEquityCurve(ctx, &paperclient.GetEquityCurveReq{UserId: input.UserID, AccountId: input.AccountID}); err == nil {
		data["equity_curve"] = reply.Items
	}
	return domain.ToolOutput{ToolName: t.Name(), Data: data, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
}
