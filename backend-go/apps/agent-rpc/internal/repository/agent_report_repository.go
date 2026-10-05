package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/common/trace"
)

type ReportRepository struct{ db *sql.DB }

func NewReportRepository(db *sql.DB) *ReportRepository { return &ReportRepository{db: db} }

type ReportInput struct {
	TaskID               uint64
	Symbol               string
	ReportType           string
	Summary              string
	MarketAnalysis       string
	TechnicalAnalysis    string
	StrategyAnalysis     string
	RagAnalysis          string
	PaperTradingAnalysis string
	RiskAnalysis         string
	FinalView            string
	RiskLevel            string
	Confidence           float64
	ReferencesJSON       string
	Disclaimer           string
	ModelConfigID        uint64
	ProviderCode         string
	Model                string
	PromptTokens         uint64
	CompletionTokens     uint64
	TotalTokens          uint64
	LatencyMs            int64
	TokenSource          string
	DataSourcesJSON      string
	EvidenceCompleteness float64
	UsageRecordID        uint64
}

func (r *ReportRepository) Create(ctx context.Context, userID uint64, report ReportInput, rawContext any, rawLLM string) (uint64, error) {
	reportJSON, _ := json.Marshal(report)
	contextJSON, _ := json.Marshal(rawContext)
	res, err := r.db.ExecContext(ctx, `
INSERT INTO agent_analysis_reports (task_id, user_id, symbol, report_type, model_config_id, provider_code, model, prompt_tokens, completion_tokens, total_tokens, latency_ms, token_source, summary, market_analysis, technical_analysis, strategy_analysis, rag_analysis, paper_trading_analysis, risk_analysis, final_view, report_json, risk_level, confidence, references_json, data_sources_json, evidence_completeness, raw_context_json, raw_llm_output, disclaimer)
VALUES (?, ?, ?, ?, NULLIF(?,0), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.TaskID, userID, report.Symbol, report.ReportType, report.ModelConfigID, report.ProviderCode, report.Model, report.PromptTokens, report.CompletionTokens, report.TotalTokens, report.LatencyMs, report.TokenSource, report.Summary, report.MarketAnalysis, report.TechnicalAnalysis, report.StrategyAnalysis, report.RagAnalysis, report.PaperTradingAnalysis, report.RiskAnalysis, report.FinalView, string(reportJSON), report.RiskLevel, report.Confidence, report.ReferencesJSON, report.DataSourcesJSON, report.EvidenceCompleteness, string(contextJSON), rawLLM, report.Disclaimer)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err == nil {
		if report.UsageRecordID > 0 {
			_, _ = r.db.ExecContext(ctx, `UPDATE model_usage_records SET report_id=? WHERE id=? AND user_id=? AND report_id IS NULL`, id, report.UsageRecordID, userID)
		} else if traceID := trace.FromContext(ctx); traceID != "" {
			_, _ = r.db.ExecContext(ctx, `UPDATE model_usage_records SET report_id=? WHERE user_id=? AND trace_id=? AND report_id IS NULL ORDER BY id DESC LIMIT 1`, id, userID, traceID)
		}
	}
	return uint64(id), err
}

func (r *ReportRepository) Get(ctx context.Context, userID, id uint64) (*agent.AgentReport, error) {
	var item agent.AgentReport
	var confidence float64
	var created time.Time
	err := r.db.QueryRowContext(ctx, `
SELECT r.id, r.task_id, t.task_no, COALESCE(r.symbol,''), r.report_type, r.summary, COALESCE(r.market_analysis,''), COALESCE(r.technical_analysis,''), COALESCE(r.strategy_analysis,''), COALESCE(r.rag_analysis,''), COALESCE(r.paper_trading_analysis,''), COALESCE(r.risk_analysis,''), COALESCE(r.final_view,''), r.confidence, r.risk_level, COALESCE(CAST(r.references_json AS CHAR),'[]'), r.disclaimer, r.created_at, COALESCE(r.model_config_id,0),r.provider_code,r.model,r.prompt_tokens,r.completion_tokens,r.total_tokens,r.latency_ms,r.token_source,COALESCE(CAST(r.data_sources_json AS CHAR),'[]'),r.evidence_completeness
FROM agent_analysis_reports r JOIN agent_tasks t ON t.id = r.task_id
WHERE r.user_id=? AND r.id=?`, userID, id).Scan(&item.Id, &item.TaskId, &item.TaskNo, &item.Symbol, &item.ReportType, &item.Summary, &item.MarketAnalysis, &item.TechnicalAnalysis, &item.StrategyAnalysis, &item.RagAnalysis, &item.PaperTradingAnalysis, &item.RiskAnalysis, &item.FinalView, &confidence, &item.RiskLevel, &item.ReferencesJson, &item.Disclaimer, &created, &item.ModelConfigId, &item.ProviderCode, &item.Model, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &item.LatencyMs, &item.TokenSource, &item.DataSourcesJson, &item.EvidenceCompleteness)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	item.Confidence = confidence
	item.CreatedAt = created.Format("2006-01-02 15:04:05")
	return &item, nil
}

func (r *ReportRepository) List(ctx context.Context, userID uint64, symbol, reportType, startDate, endDate string, limit, offset int64) ([]*agent.AgentReport, uint64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	where, args := `r.user_id=?`, []any{userID}
	if strings.TrimSpace(symbol) != "" {
		where += ` AND r.symbol=?`
		args = append(args, strings.TrimSpace(symbol))
	}
	if strings.TrimSpace(reportType) != "" {
		where += ` AND r.report_type=?`
		args = append(args, strings.TrimSpace(reportType))
	}
	if strings.TrimSpace(startDate) != "" {
		where += ` AND r.created_at>=?`
		args = append(args, strings.TrimSpace(startDate)+" 00:00:00")
	}
	if strings.TrimSpace(endDate) != "" {
		where += ` AND r.created_at<?`
		args = append(args, strings.TrimSpace(endDate)+" 23:59:59.999")
	}
	var total uint64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_analysis_reports r WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT r.id,r.task_id,t.task_no,COALESCE(r.symbol,''),r.report_type,r.summary,COALESCE(r.market_analysis,''),COALESCE(r.technical_analysis,''),COALESCE(r.strategy_analysis,''),COALESCE(r.rag_analysis,''),COALESCE(r.paper_trading_analysis,''),COALESCE(r.risk_analysis,''),COALESCE(r.final_view,''),r.confidence,r.risk_level,COALESCE(CAST(r.references_json AS CHAR),'[]'),r.disclaimer,r.created_at,COALESCE(r.model_config_id,0),r.provider_code,r.model,r.prompt_tokens,r.completion_tokens,r.total_tokens,r.latency_ms,r.token_source,COALESCE(CAST(r.data_sources_json AS CHAR),'[]'),r.evidence_completeness FROM agent_analysis_reports r JOIN agent_tasks t ON t.id=r.task_id WHERE ` + where + ` ORDER BY r.created_at DESC,r.id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]*agent.AgentReport, 0)
	for rows.Next() {
		item := &agent.AgentReport{}
		var created time.Time
		if err := rows.Scan(&item.Id, &item.TaskId, &item.TaskNo, &item.Symbol, &item.ReportType, &item.Summary, &item.MarketAnalysis, &item.TechnicalAnalysis, &item.StrategyAnalysis, &item.RagAnalysis, &item.PaperTradingAnalysis, &item.RiskAnalysis, &item.FinalView, &item.Confidence, &item.RiskLevel, &item.ReferencesJson, &item.Disclaimer, &created, &item.ModelConfigId, &item.ProviderCode, &item.Model, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &item.LatencyMs, &item.TokenSource, &item.DataSourcesJson, &item.EvidenceCompleteness); err != nil {
			return nil, 0, err
		}
		item.CreatedAt = created.Format("2006-01-02 15:04:05")
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *ReportRepository) Delete(ctx context.Context, userID uint64, ids []uint64) (uint64, error) {
	if userID == 0 || len(ids) == 0 {
		return 0, nil
	}
	if len(ids) > 200 {
		return 0, errors.New("每次最多删除 200 份分析档案")
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+1)
	args = append(args, userID)
	for _, id := range ids {
		args = append(args, id)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `UPDATE model_usage_records SET report_id=NULL WHERE user_id=? AND report_id IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE candidate_agent_reports SET agent_report_id=NULL WHERE user_id=? AND agent_report_id IN (`+placeholders+`)`, args...); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM agent_analysis_reports WHERE user_id=? AND id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	count, _ := res.RowsAffected()
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(count), nil
}
