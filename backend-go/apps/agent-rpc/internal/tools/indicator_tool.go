package tools

import (
	"context"
	"time"

	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/market-rpc/marketclient"
)

type IndicatorTool struct{ Market marketclient.Market }

func (t IndicatorTool) Name() string        { return "indicator_tool" }
func (t IndicatorTool) Description() string { return "获取技术指标与全量指标" }

func (t IndicatorTool) Execute(ctx context.Context, input domain.ToolInput) (domain.ToolOutput, error) {
	start := time.Now()
	data := map[string]any{}
	if reply, err := t.Market.Indicator(ctx, &marketclient.IndicatorRequest{Symbol: input.Symbol, Type: "macd"}); err == nil {
		data["macd"] = rawJSON(reply.Json)
	}
	if reply, err := t.Market.FullIndicator(ctx, &marketclient.IndicatorRequest{Symbol: input.Symbol}); err == nil {
		data["full"] = rawJSON(reply.Json)
	}
	return domain.ToolOutput{ToolName: t.Name(), Data: data, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
}
