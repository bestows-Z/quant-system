package types

import (
	"encoding/json"
	"strconv"
	"strings"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AddWatchlistRequest struct {
	Symbol    string `json:"symbol"`
	GroupName string `json:"group_name"`
	Remark    string `json:"remark"`
}

type CreatePaperAccountRequest struct {
	AccountName string       `json:"account_name"`
	InitialCash NumberString `json:"initial_cash"`
}

type CreatePaperOrderRequest struct {
	AccountID     uint64       `json:"account_id"`
	ClientOrderID string       `json:"client_order_id"`
	Symbol        string       `json:"symbol"`
	Name          string       `json:"name"`
	Side          string       `json:"side"`
	OrderType     string       `json:"order_type"`
	Price         NumberString `json:"price"`
	Volume        int64        `json:"volume"`
}

type CreatePortfolioRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpsertRealPositionRequest struct {
	Symbol            string       `json:"symbol"`
	Name              string       `json:"name"`
	Quantity          int64        `json:"quantity"`
	AvailableQuantity int64        `json:"available_quantity"`
	AvgCost           NumberString `json:"avg_cost"`
	OpenedOn          string       `json:"opened_on"`
	Note              string       `json:"note"`
	Version           uint32       `json:"version"`
}

type ConfirmPositionImportRequest struct {
	TaskID uint64 `json:"task_id"`
}

type SaveModelConfigRequest struct {
	ID                  uint64  `json:"id"`
	ProviderCode        string  `json:"provider_code"`
	Name                string  `json:"name"`
	BaseURL             string  `json:"base_url"`
	Model               string  `json:"model"`
	APIKey              string  `json:"api_key"`
	Temperature         float64 `json:"temperature"`
	TimeoutSeconds      uint32  `json:"timeout_seconds"`
	MaxTokens           uint32  `json:"max_tokens"`
	ContextWindowTokens uint32  `json:"context_window_tokens"`
	Enabled             bool    `json:"enabled"`
}

type CreateKnowledgeBaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type CreateDocumentRequest struct {
	KnowledgeBaseID uint64 `json:"knowledge_base_id"`
	Title           string `json:"title"`
	DocType         string `json:"doc_type"`
	SourceURL       string `json:"source_url"`
	Content         string `json:"content"`
	MetadataJSON    string `json:"metadata_json"`
}

type RagSearchRequest struct {
	KnowledgeBaseID uint64 `json:"knowledge_base_id"`
	Query           string `json:"query"`
	TopK            int64  `json:"top_k"`
}

type RagAskRequest struct {
	KnowledgeBaseID uint64 `json:"knowledge_base_id"`
	Question        string `json:"question"`
	TopK            int64  `json:"top_k"`
}

type AnalyzeStockRequest struct {
	Symbol          string `json:"symbol"`
	Question        string `json:"question"`
	KnowledgeBaseID uint64 `json:"knowledge_base_id"`
	AccountID       uint64 `json:"account_id"`
}

type AgentAskRequest struct {
	Question        string `json:"question"`
	Symbol          string `json:"symbol"`
	KnowledgeBaseID uint64 `json:"knowledge_base_id"`
	AccountID       uint64 `json:"account_id"`
}

type ExplainSignalRequest struct {
	Symbol          string `json:"symbol"`
	Question        string `json:"question"`
	KnowledgeBaseID uint64 `json:"knowledge_base_id"`
}

type ReviewBacktestRequest struct {
	BacktestID      uint64 `json:"backtest_id"`
	Question        string `json:"question"`
	KnowledgeBaseID uint64 `json:"knowledge_base_id"`
}

type ReviewPaperAccountRequest struct {
	AccountID       uint64 `json:"account_id"`
	Question        string `json:"question"`
	KnowledgeBaseID uint64 `json:"knowledge_base_id"`
}

type CreateStrategyRequest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	ParamsJSON  string `json:"params_json"`
}
type UpdateStrategyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ParamsJSON  string `json:"params_json"`
	Status      string `json:"status"`
}
type RunStrategyRequest struct {
	Symbol       string `json:"symbol"`
	StrategyCode string `json:"strategy_code"`
	ParamsJSON   string `json:"params_json"`
}
type RunBacktestRequest struct {
	Symbol              string       `json:"symbol"`
	StrategyCode        string       `json:"strategy_code"`
	StartDate           string       `json:"start_date"`
	EndDate             string       `json:"end_date"`
	InitialCash         NumberString `json:"initial_cash"`
	FeeRate             NumberString `json:"fee_rate"`
	SlippageRate        NumberString `json:"slippage_rate"`
	ParamsJSON          string       `json:"params_json"`
	Adjust              string       `json:"adjust"`
	MinCommission       NumberString `json:"min_commission"`
	StampDutyRate       NumberString `json:"stamp_duty_rate"`
	TransferFeeRate     NumberString `json:"transfer_fee_rate"`
	MaxVolumeRatio      NumberString `json:"max_volume_ratio"`
	TargetPositionRatio NumberString `json:"target_position_ratio"`
	LotSize             int64        `json:"lot_size"`
	TickSize            NumberString `json:"tick_size"`
}

type NumberString string

func (n *NumberString) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		*n = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*n = NumberString(strings.TrimSpace(text))
		return nil
	}
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return err
	}
	*n = NumberString(value)
	return nil
}

func (n NumberString) String() string {
	return string(n)
}
