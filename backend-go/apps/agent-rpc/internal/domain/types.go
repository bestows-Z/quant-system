package domain

import (
	"context"
	"encoding/json"
	"time"
)

const (
	TaskAnalyzeStock       = "ANALYZE_STOCK"
	TaskExplainSignal      = "EXPLAIN_SIGNAL"
	TaskReviewBacktest     = "REVIEW_BACKTEST"
	TaskReviewPaperAccount = "REVIEW_PAPER_ACCOUNT"
	TaskAskWithRag         = "ASK_WITH_RAG"

	StatusPending = "PENDING"
	StatusRunning = "RUNNING"
	StatusSuccess = "SUCCESS"
	StatusFailed  = "FAILED"

	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"

	RiskLow     = "LOW"
	RiskMedium  = "MEDIUM"
	RiskHigh    = "HIGH"
	RiskExtreme = "EXTREME"

	Disclaimer = "仅供研究，不构成投资建议"
)

type ToolInput struct {
	UserID          uint64         `json:"user_id"`
	Symbol          string         `json:"symbol"`
	Question        string         `json:"question"`
	AccountID       uint64         `json:"account_id"`
	KnowledgeBaseID uint64         `json:"knowledge_base_id"`
	Extra           map[string]any `json:"extra,omitempty"`
}

type ToolOutput struct {
	ToolName  string `json:"tool_name"`
	Data      any    `json:"data"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
	Success   bool   `json:"success"`
}

type AgentTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input ToolInput) (ToolOutput, error)
}

type AnalyzeRequest struct {
	UserID          uint64
	TaskType        string
	Symbol          string
	Question        string
	AccountID       uint64
	KnowledgeBaseID uint64
	Title           string
	Extra           map[string]any
}

type AnalysisReport struct {
	ID                   uint64          `json:"id,omitempty"`
	TaskID               uint64          `json:"task_id,omitempty"`
	TaskNo               string          `json:"task_no,omitempty"`
	Symbol               string          `json:"symbol,omitempty"`
	ReportType           string          `json:"report_type"`
	Summary              string          `json:"summary"`
	MarketAnalysis       string          `json:"market_analysis"`
	TechnicalAnalysis    string          `json:"technical_analysis"`
	StrategyAnalysis     string          `json:"strategy_analysis"`
	RagAnalysis          string          `json:"rag_analysis"`
	PaperTradingAnalysis string          `json:"paper_trading_analysis"`
	RiskAnalysis         string          `json:"risk_analysis"`
	FinalView            string          `json:"final_view"`
	Confidence           float64         `json:"confidence"`
	RiskLevel            string          `json:"risk_level"`
	References           json.RawMessage `json:"references"`
	Disclaimer           string          `json:"disclaimer"`
	CreatedAt            string          `json:"created_at,omitempty"`
}

type AgentContext struct {
	Request AnalyzeRequest        `json:"request"`
	Tools   map[string]ToolOutput `json:"tools"`
}

type ChatRequest struct {
	Prompt  string       `json:"prompt"`
	Context AgentContext `json:"context"`
}

type ChatResponse struct {
	Content          string
	Report           AnalysisReport
	Provider         string
	Model            string
	LatencyMs        int64
	RawCreated       time.Time
	ConfigID         uint64
	PromptTokens     uint64
	CompletionTokens uint64
	TotalTokens      uint64
	TokenSource      string
	UsageRecordID    uint64
}

type LLMClient interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
