package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatService struct {
	repo         *repository.ChatRepository
	orchestrator *Orchestrator
}

func NewChatService(repo *repository.ChatRepository, orchestrator *Orchestrator) *ChatService {
	return &ChatService{repo: repo, orchestrator: orchestrator}
}

func (s *ChatService) CreateSession(ctx context.Context, userID uint64, title string) (*agent.AgentChatSession, error) {
	return s.repo.CreateSession(ctx, userID, title)
}
func (s *ChatService) ListSessions(ctx context.Context, userID uint64) ([]*agent.AgentChatSession, error) {
	return s.repo.ListSessions(ctx, userID)
}
func (s *ChatService) RenameSession(ctx context.Context, userID, sessionID uint64, title string) (*agent.AgentChatSession, error) {
	return s.repo.RenameSession(ctx, userID, sessionID, title)
}
func (s *ChatService) DeleteSession(ctx context.Context, userID, sessionID uint64) (uint64, error) {
	return s.repo.DeleteSession(ctx, userID, sessionID)
}
func (s *ChatService) ClearMessages(ctx context.Context, userID, sessionID uint64) (uint64, error) {
	return s.repo.ClearMessages(ctx, userID, sessionID)
}
func (s *ChatService) ListMessages(ctx context.Context, userID, sessionID uint64) ([]*agent.AgentChatMessage, error) {
	return s.repo.ListMessages(ctx, userID, sessionID)
}

func (s *ChatService) Ask(ctx context.Context, in *agent.AgentChatAskReq) (*agent.AgentChatAskResp, error) {
	content := strings.TrimSpace(in.GetContent())
	if in.GetUserId() == 0 || in.GetSessionId() == 0 || strings.TrimSpace(in.GetRequestId()) == "" || content == "" {
		return nil, status.Error(codes.InvalidArgument, "聊天请求参数不完整")
	}
	if len([]rune(content)) > 4000 {
		return nil, status.Error(codes.InvalidArgument, "单条问题不能超过 4000 个字符")
	}
	if existing, err := s.repo.FindAssistant(ctx, in.GetUserId(), in.GetSessionId(), in.GetRequestId()); err != nil {
		return nil, err
	} else if existing != nil && existing.Status == "COMPLETED" {
		return &agent.AgentChatAskResp{Message: existing}, nil
	}
	session, err := s.repo.GetSession(ctx, in.GetUserId(), in.GetSessionId())
	if err != nil {
		return nil, err
	}
	_, err = s.repo.InsertMessage(ctx, in.GetUserId(), in.GetSessionId(), &agent.AgentChatMessage{RequestId: in.GetRequestId(), Role: "user", Content: content, Status: "COMPLETED"})
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return nil, err
	}
	if session.Title == "新研究会话" {
		titleRunes := []rune(content)
		if len(titleRunes) > 24 {
			titleRunes = titleRunes[:24]
		}
		_, _ = s.repo.RenameSession(ctx, in.GetUserId(), in.GetSessionId(), string(titleRunes))
	}
	started := time.Now()
	result, err := s.orchestrator.Ask(ctx, AnalyzeRequest{UserID: in.GetUserId(), Symbol: strings.TrimSpace(in.GetSymbol()), Question: content, AccountID: in.GetAccountId(), KnowledgeBaseID: in.GetKnowledgeBaseId()})
	if err != nil {
		failed, _ := s.repo.InsertMessage(context.WithoutCancel(ctx), in.GetUserId(), in.GetSessionId(), &agent.AgentChatMessage{RequestId: in.GetRequestId(), Role: "assistant", Content: "研究分析暂时失败，请稍后重试。", Status: "FAILED", ErrorMessage: err.Error(), LatencyMs: time.Since(started).Milliseconds()})
		return &agent.AgentChatAskResp{Message: failed}, err
	}
	report, _ := s.orchestrator.GetReport(ctx, in.GetUserId(), result.GetReportId())
	answer := formatChatAnswer(result)
	toolCalls, _ := json.Marshal([]map[string]any{{"name": "quant_research_pipeline", "mode": "READ_ONLY", "report_id": result.GetReportId()}})
	message := &agent.AgentChatMessage{RequestId: in.GetRequestId(), Role: "assistant", Content: answer, Status: "COMPLETED", LatencyMs: time.Since(started).Milliseconds(), CitationsJson: result.GetReferencesJson(), ToolCallsJson: string(toolCalls)}
	if report != nil {
		message.Provider = report.GetProviderCode()
		message.Model = report.GetModel()
		message.InputTokens = int32(report.GetPromptTokens())
		message.OutputTokens = int32(report.GetCompletionTokens())
	}
	message, err = s.repo.InsertMessage(ctx, in.GetUserId(), in.GetSessionId(), message)
	if err != nil {
		return nil, err
	}
	return &agent.AgentChatAskResp{Message: message, ReportId: result.GetReportId()}, nil
}

func formatChatAnswer(result *agent.AnalyzeStockResp) string {
	sections := []struct{ title, body string }{
		{"结论摘要", result.GetSummary()}, {"市场与行情", result.GetMarketAnalysis()},
		{"技术与量价", result.GetTechnicalAnalysis()}, {"策略分析", result.GetStrategyAnalysis()},
		{"知识与事件", result.GetRagAnalysis()}, {"持仓与模拟交易", result.GetPaperTradingAnalysis()},
		{"风险提示", result.GetRiskAnalysis()}, {"综合判断", result.GetFinalView()},
	}
	parts := make([]string, 0, len(sections)+1)
	for _, section := range sections {
		if body := strings.TrimSpace(section.body); body != "" {
			parts = append(parts, fmt.Sprintf("## %s\n%s", section.title, body))
		}
	}
	parts = append(parts, "仅供研究，不构成投资建议。")
	return strings.Join(parts, "\n\n")
}
