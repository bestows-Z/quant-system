package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/repository"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"
	"quant-system/backend-go/common/xmq"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Orchestrator struct {
	tasks    *repository.TaskRepository
	messages *repository.MessageRepository
	calls    *repository.ToolCallRepository
	reports  *repository.ReportRepository
	tools    []AgentTool
	llm      LLMClient
	producer xmq.Producer
}

func NewOrchestrator(tasks *repository.TaskRepository, messages *repository.MessageRepository, calls *repository.ToolCallRepository, reports *repository.ReportRepository, tools []AgentTool, llm LLMClient, producer xmq.Producer) *Orchestrator {
	return &Orchestrator{tasks: tasks, messages: messages, calls: calls, reports: reports, tools: tools, llm: llm, producer: producer}
}

func (o *Orchestrator) AnalyzeStock(ctx context.Context, req AnalyzeRequest) (*agent.AnalyzeStockResp, error) {
	if req.UserID == 0 {
		return nil, status.Error(codes.Unauthenticated, "invalid user")
	}
	if req.TaskType == "" {
		req.TaskType = TaskAnalyzeStock
	}
	if req.TaskType == TaskAnalyzeStock && strings.TrimSpace(req.Symbol) == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol 不能为空")
	}
	if strings.TrimSpace(req.Question) == "" {
		req.Question = "请基于当前上下文进行量化研究分析"
	}
	taskNo := "AT" + idgen.NewEventID()
	taskID, err := o.tasks.Create(ctx, taskNo, req.UserID, req.TaskType, req.Symbol, req.Title, req.Question, req)
	if err != nil {
		return nil, err
	}
	o.publish(ctx, constants.EventAgentTaskCreated, req.UserID, map[string]any{"task_id": taskID, "task_no": taskNo, "task_type": req.TaskType})
	_ = o.messages.Create(ctx, taskID, req.UserID, RoleUser, req.Question)
	if err := o.tasks.MarkRunning(ctx, taskID, req.UserID); err != nil {
		return nil, err
	}
	input := ToolInput{UserID: req.UserID, Symbol: req.Symbol, Question: req.Question, AccountID: req.AccountID, KnowledgeBaseID: req.KnowledgeBaseID, Extra: req.Extra}
	outputs := make(map[string]ToolOutput)
	activeTools := make([]AgentTool, 0, len(o.tools))
	for _, tool := range o.tools {
		if req.AccountID == 0 && tool.Name() == "paper_account_tool" {
			continue
		}
		activeTools = append(activeTools, tool)
	}
	toolOutputs := ExecuteTools(ctx, activeTools, input)
	for i, tool := range activeTools {
		out := toolOutputs[i]
		outputs[tool.Name()] = out
		callStatus := "SUCCESS"
		if !out.Success {
			callStatus = "FAILED"
		}
		_ = o.calls.Create(ctx, taskID, req.UserID, tool.Name(), input, out, callStatus, out.Error, out.LatencyMs)
	}
	agentCtx := ContextBuilder{}.Build(req, outputs)
	prompt := PromptBuilder{}.Build(agentCtx)
	llmResp, err := o.llm.Chat(ctx, ChatRequest{Prompt: prompt, Context: agentCtx})
	if err != nil {
		o.markFailed(taskID, req.UserID, err)
		o.publish(context.Background(), constants.EventAgentTaskFailed, req.UserID, map[string]any{"task_id": taskID, "error": err.Error()})
		return nil, status.Error(codes.Internal, "LLM 调用失败")
	}
	report := ReportBuilder{}.FromLLM(taskID, taskNo, req, agentCtx, llmResp)
	dataSources, completeness := evidenceMetadata(agentCtx)
	tokenSource := llmResp.TokenSource
	if tokenSource == "" {
		if llmResp.Provider == "local-rules" {
			tokenSource = "LOCAL"
		} else {
			tokenSource = "PROVIDER"
		}
	}
	reportInput := repository.ReportInput{TaskID: taskID, Symbol: report.Symbol, ReportType: report.ReportType, Summary: report.Summary, MarketAnalysis: report.MarketAnalysis, TechnicalAnalysis: report.TechnicalAnalysis, StrategyAnalysis: report.StrategyAnalysis, RagAnalysis: report.RagAnalysis, PaperTradingAnalysis: report.PaperTradingAnalysis, RiskAnalysis: report.RiskAnalysis, FinalView: report.FinalView, RiskLevel: report.RiskLevel, Confidence: report.Confidence, ReferencesJSON: string(report.References), Disclaimer: report.Disclaimer, ModelConfigID: llmResp.ConfigID, ProviderCode: llmResp.Provider, Model: llmResp.Model, PromptTokens: llmResp.PromptTokens, CompletionTokens: llmResp.CompletionTokens, TotalTokens: llmResp.TotalTokens, LatencyMs: llmResp.LatencyMs, TokenSource: tokenSource, DataSourcesJSON: dataSources, EvidenceCompleteness: completeness, UsageRecordID: llmResp.UsageRecordID}
	reportID, err := o.reports.Create(ctx, req.UserID, reportInput, agentCtx, llmResp.Content)
	if err != nil {
		o.markFailed(taskID, req.UserID, err)
		o.publish(context.Background(), constants.EventAgentTaskFailed, req.UserID, map[string]any{"task_id": taskID, "error": err.Error()})
		return nil, err
	}
	report.ID = reportID
	if err := o.tasks.MarkSuccess(ctx, taskID, req.UserID, report.RiskLevel, report.Confidence, report.Summary, report); err != nil {
		return nil, err
	}
	assistantMsg, _ := json.Marshal(report)
	_ = o.messages.Create(ctx, taskID, req.UserID, RoleAssistant, string(assistantMsg))
	o.publish(ctx, constants.EventAgentTaskCompleted, req.UserID, map[string]any{"task_id": taskID, "report_id": reportID, "risk_level": report.RiskLevel})
	response := reportResp(report, taskID, taskNo, reportID)
	response.ModelConfigId = reportInput.ModelConfigID
	response.ProviderCode = reportInput.ProviderCode
	response.Model = reportInput.Model
	response.PromptTokens = reportInput.PromptTokens
	response.CompletionTokens = reportInput.CompletionTokens
	response.TotalTokens = reportInput.TotalTokens
	response.LatencyMs = reportInput.LatencyMs
	response.TokenSource = reportInput.TokenSource
	response.DataSourcesJson = reportInput.DataSourcesJSON
	response.EvidenceCompleteness = reportInput.EvidenceCompleteness
	return response, nil
}

func (o *Orchestrator) markFailed(taskID, userID uint64, cause error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = o.tasks.MarkFailed(ctx, taskID, userID, cause.Error())
}

func (o *Orchestrator) Ask(ctx context.Context, req AnalyzeRequest) (*agent.AnalyzeStockResp, error) {
	req.TaskType = TaskAskWithRag
	req.Title = "Agent Ask"
	return o.AnalyzeStock(ctx, req)
}

func (o *Orchestrator) ListTasks(ctx context.Context, userID uint64, statusText, taskType string, limit int64) ([]*agent.AgentTask, error) {
	return o.tasks.List(ctx, userID, statusText, taskType, limit)
}

func (o *Orchestrator) GetTask(ctx context.Context, userID, id uint64) (*agent.AgentTask, error) {
	item, err := o.tasks.Get(ctx, userID, id)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "Agent 任务不存在")
	}
	return item, err
}

func (o *Orchestrator) GetReport(ctx context.Context, userID, id uint64) (*agent.AgentReport, error) {
	item, err := o.reports.Get(ctx, userID, id)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "Agent 报告不存在")
	}
	return item, err
}

func (o *Orchestrator) ListReports(ctx context.Context, in *agent.ListAgentReportsReq) ([]*agent.AgentReport, uint64, error) {
	return o.reports.List(ctx, in.GetUserId(), in.GetSymbol(), in.GetReportType(), in.GetStartDate(), in.GetEndDate(), in.GetLimit(), in.GetOffset())
}

func (o *Orchestrator) DeleteReports(ctx context.Context, userID uint64, ids []uint64) (uint64, error) {
	return o.reports.Delete(ctx, userID, ids)
}

func (o *Orchestrator) publish(ctx context.Context, eventType string, userID uint64, payload map[string]any) {
	if o.producer == nil {
		return
	}
	_ = o.producer.Publish(ctx, xmq.Event{EventID: idgen.NewEventID(), EventType: eventType, TraceID: trace.FromContext(ctx), UserID: userID, Payload: payload, CreatedAt: time.Now()})
}

func reportResp(r AnalysisReport, taskID uint64, taskNo string, reportID uint64) *agent.AnalyzeStockResp {
	return &agent.AnalyzeStockResp{TaskId: taskID, TaskNo: taskNo, Symbol: r.Symbol, Summary: r.Summary, MarketAnalysis: r.MarketAnalysis, TechnicalAnalysis: r.TechnicalAnalysis, StrategyAnalysis: r.StrategyAnalysis, RagAnalysis: r.RagAnalysis, PaperTradingAnalysis: r.PaperTradingAnalysis, RiskAnalysis: r.RiskAnalysis, FinalView: r.FinalView, Confidence: r.Confidence, RiskLevel: r.RiskLevel, ReferencesJson: string(r.References), Disclaimer: r.Disclaimer, ReportId: reportID}
}

func evidenceMetadata(ctx AgentContext) (string, float64) {
	items := make([]map[string]any, 0, len(ctx.Tools))
	success := 0
	for name, output := range ctx.Tools {
		if output.Success {
			success++
		}
		items = append(items, map[string]any{"tool": name, "success": output.Success, "latency_ms": output.LatencyMs, "error": output.Error})
	}
	raw, _ := json.Marshal(items)
	if len(ctx.Tools) == 0 {
		return string(raw), 0
	}
	return string(raw), float64(success) / float64(len(ctx.Tools))
}

func DebugJSON(v any) string {
	b, _ := json.Marshal(v)
	return fmt.Sprintf("%s", b)
}
