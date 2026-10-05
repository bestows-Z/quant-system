package tools

import (
	"context"
	"time"

	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/rag-rpc/ragclient"
)

type RagSearchTool struct{ Rag ragclient.Rag }

func (t RagSearchTool) Name() string        { return "rag_search_tool" }
func (t RagSearchTool) Description() string { return "调用 rag-rpc 检索知识库上下文" }

func (t RagSearchTool) Execute(ctx context.Context, input domain.ToolInput) (domain.ToolOutput, error) {
	start := time.Now()
	if input.Question == "" {
		return domain.ToolOutput{ToolName: t.Name(), Data: map[string]any{"message": "分析问题为空，未执行知识检索"}, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
	}
	reply, err := t.Rag.Search(ctx, &ragclient.SearchReq{UserId: input.UserID, KnowledgeBaseId: input.KnowledgeBaseID, Query: input.Question, TopK: 5})
	if err != nil {
		return domain.ToolOutput{ToolName: t.Name(), Error: err.Error(), Success: false, LatencyMs: time.Since(start).Milliseconds()}, err
	}
	return domain.ToolOutput{ToolName: t.Name(), Data: reply, Success: true, LatencyMs: time.Since(start).Milliseconds()}, nil
}
