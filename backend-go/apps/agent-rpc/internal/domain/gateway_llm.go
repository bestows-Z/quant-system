package domain

import (
	"context"
	"encoding/json"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgatewayservice"
	"time"
)

type GatewayLLMClient struct {
	Gateway  modelgatewayservice.ModelGatewayService
	Fallback LLMClient
}

func (c GatewayLLMClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	reply, err := c.Gateway.Generate(ctx, &modelgatewayservice.GenerateReq{UserId: req.Context.Request.UserID, SystemPrompt: "你是严谨的量化研究报告引擎。严格遵循用户消息中的 JSON 字段定义，只依据工具数据生成报告；缺少证据时明确说明数据不足。", Prompt: req.Prompt, ResponseFormat: "json_object", RequestType: req.Context.Request.TaskType})
	if err != nil {
		return c.Fallback.Chat(ctx, req)
	}
	content := stripJSONFence(reply.Content)
	var report AnalysisReport
	if json.Unmarshal([]byte(content), &report) != nil {
		return c.Fallback.Chat(ctx, req)
	}
	return ChatResponse{Content: content, Report: report, Provider: reply.ProviderCode, Model: reply.Model, LatencyMs: reply.LatencyMs, RawCreated: time.Now(), ConfigID: reply.ConfigId, PromptTokens: uint64(reply.PromptTokens), CompletionTokens: uint64(reply.CompletionTokens), TotalTokens: uint64(reply.TotalTokens), TokenSource: reply.TokenSource, UsageRecordID: reply.UsageRecordId}, nil
}
func NewGatewayLLM(g modelgatewayservice.ModelGatewayService) LLMClient {
	return GatewayLLMClient{Gateway: g, Fallback: RulesLLMClient{Provider: "local-rules", Model: "quant-tool-synthesis-v1"}}
}
