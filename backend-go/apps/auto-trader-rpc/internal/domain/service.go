package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"quant-system/backend-go/apps/auto-trader-rpc/autotrader"
	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/feature-rpc/featureservice"
	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgatewayservice"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/risk-rpc/riskclient"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/markettime"
	"quant-system/backend-go/common/strategyparams"
	"quant-system/backend-go/common/trace"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const disclaimer = "AI 自动交易仅用于模拟交易研究，不连接券商，不构成投资建议"

type Service struct {
	db       *sql.DB
	feature  featureservice.FeatureService
	strategy strategyclient.Strategy
	risk     riskclient.Risk
	paper    paperclient.Paper
	market   marketclient.Market
	model    modelgatewayservice.ModelGatewayService
	backtest backtestclient.Backtest
	runMu    sync.Mutex
}

func NewService(db *sql.DB, feature featureservice.FeatureService, strategy strategyclient.Strategy, risk riskclient.Risk, paper paperclient.Paper, market marketclient.Market, model modelgatewayservice.ModelGatewayService, backtest backtestclient.Backtest) *Service {
	return &Service{db: db, feature: feature, strategy: strategy, risk: risk, paper: paper, market: market, model: model, backtest: backtest}
}

type account struct {
	ID                    uint64  `json:"id"`
	UserID                uint64  `json:"user_id"`
	PaperAccountID        uint64  `json:"paper_account_id"`
	Name                  string  `json:"name"`
	Status                string  `json:"status"`
	InitialCapital        float64 `json:"initial_capital"`
	MaxPositionRatio      float64 `json:"max_position_ratio"`
	MaxTotalPositionRatio float64 `json:"max_total_position_ratio"`
	MaxDailyLossRatio     float64 `json:"max_daily_loss_ratio"`
	MinSignalScore        float64 `json:"min_signal_score"`
	MaxSymbols            int32   `json:"max_symbols"`
	CreatedAt             string  `json:"created_at"`
	ArchivedAt            string  `json:"archived_at,omitempty"`
}

type strategyVersion struct {
	ID            uint64              `json:"id"`
	AutoAccountID uint64              `json:"auto_account_id"`
	VersionNo     int32               `json:"version_no"`
	Name          string              `json:"name"`
	BaseStrategy  string              `json:"base_strategy"`
	Config        map[string]any      `json:"config"`
	Source        string              `json:"source"`
	Lifecycle     string              `json:"lifecycle"`
	ModelProvider string              `json:"model_provider"`
	ModelName     string              `json:"model_name"`
	Rationale     string              `json:"rationale"`
	CreatedAt     string              `json:"created_at"`
	Evaluation    *strategyEvaluation `json:"evaluation,omitempty"`
}

type strategyEvaluation struct {
	ID                uint64           `json:"id"`
	TestedSymbols     int              `json:"tested_symbols"`
	SuccessfulSymbols int              `json:"successful_symbols"`
	TotalTrades       int64            `json:"total_trades"`
	AnnualReturn      float64          `json:"annual_return"`
	MaxDrawdown       float64          `json:"max_drawdown"`
	WinRate           float64          `json:"win_rate"`
	Passed            bool             `json:"passed"`
	StartDate         string           `json:"start_date"`
	EndDate           string           `json:"end_date"`
	Evidence          []map[string]any `json:"evidence"`
	CreatedAt         string           `json:"created_at"`
}

type candidate struct {
	Symbol     string         `json:"symbol"`
	Name       string         `json:"name"`
	Score      float64        `json:"total_score"`
	Confidence float64        `json:"confidence"`
	DataMode   string         `json:"data_mode"`
	Evidence   map[string]any `json:"evidence"`
}
type bookLevel struct {
	Level  int32   `json:"level"`
	Price  float64 `json:"price"`
	Volume int64   `json:"volume"`
}
type book struct {
	Symbol    string      `json:"symbol"`
	Name      string      `json:"name"`
	Price     float64     `json:"price"`
	QuoteTime string      `json:"quote_time"`
	DataMode  string      `json:"data_mode"`
	Bids      []bookLevel `json:"bids"`
	Asks      []bookLevel `json:"asks"`
}

func (s *Service) CreateAccount(ctx context.Context, in *autotrader.CreateAccountReq) (string, error) {
	if in.GetUserId() == 0 || in.GetPaperAccountId() == 0 || strings.TrimSpace(in.GetName()) == "" {
		return "", status.Error(codes.InvalidArgument, "账户参数不完整")
	}
	assets, err := s.paper.GetAccountAssets(ctx, &paperclient.GetAccountAssetsReq{UserId: in.GetUserId(), AccountId: in.GetPaperAccountId()})
	if err != nil || assets.GetAccount() == nil {
		return "", status.Error(codes.FailedPrecondition, "模拟账户不存在或无权访问")
	}
	positionRatio := clamp(in.GetMaxPositionRatio(), 0.01, 0.3, 0.1)
	dailyLoss := clamp(in.GetMaxDailyLossRatio(), 0.005, 0.1, 0.02)
	minScore := clamp(in.GetMinSignalScore(), 50, 100, 70)
	maxSymbols := in.GetMaxSymbols()
	if maxSymbols <= 0 || maxSymbols > 50 {
		maxSymbols = 10
	}
	maxTotalRatio := clamp(in.GetMaxTotalPositionRatio(), 0.1, 1, 0.8)
	if maxTotalRatio < positionRatio {
		maxTotalRatio = positionRatio
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO auto_trade_accounts(user_id,paper_account_id,name,status,initial_capital,max_position_ratio,max_total_position_ratio,max_daily_loss_ratio,min_signal_score,max_symbols) VALUES(?,?,?,'PAUSED',?,?,?,?,?,?)`, in.GetUserId(), in.GetPaperAccountId(), strings.TrimSpace(in.GetName()), assets.Account.InitialCash, positionRatio, maxTotalRatio, dailyLoss, minScore, maxSymbols)
	if err != nil {
		return "", err
	}
	id, _ := res.LastInsertId()
	item, err := s.getAccount(ctx, in.GetUserId(), uint64(id))
	if err != nil {
		return "", err
	}
	return envelope(item), nil
}

func (s *Service) ListAccounts(ctx context.Context, userID uint64) (string, error) {
	return s.listAccounts(ctx, userID, false)
}

func (s *Service) ListArchivedAccounts(ctx context.Context, userID uint64) (string, error) {
	return s.listAccounts(ctx, userID, true)
}

func (s *Service) listAccounts(ctx context.Context, userID uint64, archived bool) (string, error) {
	condition := "deleted_at IS NULL"
	if archived {
		condition = "archived_at IS NOT NULL"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,user_id,paper_account_id,name,status,initial_capital,max_position_ratio,max_total_position_ratio,max_daily_loss_ratio,min_signal_score,max_symbols,created_at,archived_at FROM auto_trade_accounts WHERE user_id=? AND `+condition+` ORDER BY id DESC`, userID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	items := make([]account, 0)
	for rows.Next() {
		item, err := scanAccount(rows)
		if err != nil {
			return "", err
		}
		items = append(items, item)
	}
	return envelope(map[string]any{"items": items, "disclaimer": disclaimer}), rows.Err()
}

func (s *Service) UpdateAccount(ctx context.Context, in *autotrader.UpdateAccountReq) (string, error) {
	acc, err := s.getAccount(ctx, in.GetUserId(), in.GetAccountId())
	if err != nil {
		return "", err
	}
	if acc.Status != "PAUSED" {
		return "", status.Error(codes.FailedPrecondition, "请先暂停自动交易，再编辑账户")
	}
	name := strings.TrimSpace(in.GetName())
	if name == "" {
		return "", status.Error(codes.InvalidArgument, "请输入账户名称")
	}
	positionRatio := clamp(in.GetMaxPositionRatio(), 0.01, 0.3, acc.MaxPositionRatio)
	totalRatio := clamp(in.GetMaxTotalPositionRatio(), 0.1, 1, acc.MaxTotalPositionRatio)
	if totalRatio < positionRatio {
		totalRatio = positionRatio
	}
	dailyLoss := clamp(in.GetMaxDailyLossRatio(), 0.005, 0.1, acc.MaxDailyLossRatio)
	minScore := clamp(in.GetMinSignalScore(), 50, 100, acc.MinSignalScore)
	maxSymbols := in.GetMaxSymbols()
	if maxSymbols <= 0 || maxSymbols > 50 {
		maxSymbols = acc.MaxSymbols
	}
	_, err = s.db.ExecContext(ctx, `UPDATE auto_trade_accounts SET name=?,max_position_ratio=?,max_total_position_ratio=?,max_daily_loss_ratio=?,min_signal_score=?,max_symbols=? WHERE user_id=? AND id=? AND status='PAUSED' AND deleted_at IS NULL`, name, positionRatio, totalRatio, dailyLoss, minScore, maxSymbols, in.GetUserId(), in.GetAccountId())
	if err != nil {
		return "", err
	}
	updated, err := s.getAccount(ctx, in.GetUserId(), in.GetAccountId())
	if err != nil {
		return "", err
	}
	return envelope(updated), nil
}

func (s *Service) ArchiveAccount(ctx context.Context, userID, accountID uint64) (string, error) {
	var state string
	var archived sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT status,archived_at FROM auto_trade_accounts WHERE user_id=? AND id=?`, userID, accountID).Scan(&state, &archived)
	if errors.Is(err, sql.ErrNoRows) {
		return envelope(map[string]any{"id": accountID, "already_archived": true}), nil
	}
	if err != nil {
		return "", err
	}
	if archived.Valid || state == "ARCHIVED" {
		return envelope(map[string]any{"id": accountID, "already_archived": true}), nil
	}
	if state != "PAUSED" {
		return "", status.Error(codes.FailedPrecondition, "请先暂停自动交易，再归档账户")
	}
	_, err = s.db.ExecContext(ctx, `UPDATE auto_trade_accounts SET status='ARCHIVED',archived_at=CURRENT_TIMESTAMP,deleted_at=CURRENT_TIMESTAMP WHERE user_id=? AND id=? AND status='PAUSED'`, userID, accountID)
	if err != nil {
		return "", err
	}
	return envelope(map[string]any{"id": accountID, "status": "ARCHIVED", "already_archived": false}), nil
}

func (s *Service) GenerateStrategy(ctx context.Context, in *autotrader.GenerateStrategyReq) (string, error) {
	acc, err := s.getAccount(ctx, in.GetUserId(), in.GetAccountId())
	if err != nil {
		return "", err
	}
	objective := strings.TrimSpace(in.GetObjective())
	if objective == "" {
		return "", status.Error(codes.InvalidArgument, "请填写 AI 生成研究目标，或选择一个目标模板")
	}
	config := map[string]any{"base_strategy": "trend_score", "params": map[string]any{}, "objective": objective}
	source, provider, modelName, rationale := "RULE", "", "", "使用固定趋势评分基线；模型不可用或输出未通过参数校验。"
	if generated, genErr := s.model.Generate(ctx, &modelgatewayservice.GenerateReq{UserId: in.GetUserId(), SystemPrompt: "你是量化策略配置器。只选择 trend_score、ma_cross、macd_cross、rsi_reversal、boll_breakout、volume_breakout。返回严格 JSON：{base_strategy,params,rationale}。参数仅允许：ma_cross 为 short_window/long_window；macd_cross 为 fast/slow/signal；rsi_reversal 为 period/low/high；boll_breakout 为 period/std；volume_breakout 为 volume_window/volume_multiplier；trend_score 的 params 必须为空对象。所有参数必须是范围合理的数字，不得输出代码或收益承诺。", Prompt: fmt.Sprintf("目标：%s；单票上限 %.2f%%；最低信号分 %.1f。", objective, acc.MaxPositionRatio*100, acc.MinSignalScore), ResponseFormat: "json", RequestType: "auto_strategy_generate"}); genErr == nil {
		if parsed, ok := parseGeneratedConfig(generated.GetContent()); ok {
			config = parsed
			source = "AI"
			provider = generated.GetProviderCode()
			modelName = generated.GetModel()
			rationale = stringValue(parsed["rationale"], "AI 已生成受约束策略配置")
		}
	}
	base := stringValue(config["base_strategy"], "trend_score")
	if _, _, err := effectiveStrategyConfig(strategyVersion{BaseStrategy: base, Config: config}); err != nil {
		return "", status.Error(codes.InvalidArgument, "模型生成了不受支持的策略")
	}
	configJSON, _ := json.Marshal(config)
	var version int32
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version_no),0)+1 FROM auto_strategy_versions WHERE auto_account_id=?`, acc.ID).Scan(&version); err != nil {
		return "", err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO auto_strategy_versions(auto_account_id,user_id,version_no,name,base_strategy,config_json,source,lifecycle,model_provider,model_name,rationale) VALUES(?,?,?,?,?,?,'`+source+`','SHADOW',?,?,?)`, acc.ID, in.GetUserId(), version, fmt.Sprintf("%s v%d", strategyLabel(base), version), base, string(configJSON), provider, modelName, rationale)
	if err != nil {
		return "", err
	}
	id, _ := res.LastInsertId()
	item, err := s.getStrategy(ctx, in.GetUserId(), uint64(id))
	if err != nil {
		return "", err
	}
	return envelope(map[string]any{"item": item, "notice": "策略已进入影子阶段；查看配置并明确晋级后才允许模拟执行", "disclaimer": disclaimer}), nil
}

func (s *Service) ListStrategies(ctx context.Context, userID, accountID uint64) (string, error) {
	if _, err := s.getAccount(ctx, userID, accountID); err != nil {
		return "", err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,auto_account_id,version_no,name,base_strategy,CAST(config_json AS CHAR),source,lifecycle,model_provider,model_name,COALESCE(rationale,''),created_at FROM auto_strategy_versions WHERE user_id=? AND auto_account_id=? ORDER BY version_no DESC`, userID, accountID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	items := make([]strategyVersion, 0)
	for rows.Next() {
		item, err := scanStrategy(rows)
		if err != nil {
			return "", err
		}
		item.Evaluation, _ = s.latestEvaluation(ctx, item.ID)
		items = append(items, item)
	}
	return envelope(map[string]any{"items": items, "disclaimer": disclaimer}), rows.Err()
}

func (s *Service) GetStrategy(ctx context.Context, userID, strategyID uint64) (string, error) {
	item, err := s.getStrategy(ctx, userID, strategyID)
	if err != nil {
		return "", err
	}
	item.Evaluation, _ = s.latestEvaluation(ctx, item.ID)
	return envelope(item), nil
}

func (s *Service) EvaluateStrategy(ctx context.Context, userID, strategyID uint64) (string, error) {
	item, err := s.getStrategy(ctx, userID, strategyID)
	if err != nil {
		return "", err
	}
	acc, err := s.getAccount(ctx, userID, item.AutoAccountID)
	if err != nil {
		return "", err
	}
	evaluation, err := s.evaluateStrategy(ctx, acc, item)
	if err != nil {
		return "", err
	}
	return envelope(map[string]any{"evaluation": evaluation, "notice": evaluationNotice(evaluation), "disclaimer": disclaimer}), nil
}

func (s *Service) PromoteStrategy(ctx context.Context, userID, strategyID uint64) (string, error) {
	item, err := s.getStrategy(ctx, userID, strategyID)
	if err != nil {
		return "", err
	}
	evaluation, err := s.latestEvaluation(ctx, item.ID)
	if err != nil || evaluation == nil {
		return "", status.Error(codes.FailedPrecondition, "策略尚未完成真实历史回测，请先点击评估")
	}
	if !evaluation.Passed {
		return "", status.Error(codes.FailedPrecondition, "策略未通过回测门槛，不能晋级；请查看评估证据并调整策略")
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE auto_strategy_versions SET lifecycle='RETIRED' WHERE user_id=? AND auto_account_id=? AND lifecycle='PROMOTED'`, userID, item.AutoAccountID); err != nil {
		return "", err
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE auto_strategy_versions SET lifecycle='PROMOTED' WHERE user_id=? AND id=?`, userID, strategyID); err != nil {
		return "", err
	}
	item, err = s.getStrategy(ctx, userID, strategyID)
	if err != nil {
		return "", err
	}
	item.Evaluation = evaluation
	return envelope(map[string]any{"item": item, "notice": "策略已晋级；启用账户后仅在交易时段、真实盘口新鲜且通过风控时执行模拟限价单", "disclaimer": disclaimer}), nil
}

func (s *Service) Start(ctx context.Context, userID, accountID uint64) (string, error) {
	acc, err := s.getAccount(ctx, userID, accountID)
	if err != nil {
		return "", err
	}
	st, err := s.promotedStrategy(ctx, userID, accountID)
	if err != nil {
		return "", status.Error(codes.FailedPrecondition, "请先生成、检查并晋级一个策略")
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE auto_trade_accounts SET status='ACTIVE' WHERE user_id=? AND id=?`, userID, accountID); err != nil {
		return "", err
	}
	result, err := s.runAccountOnce(ctx, acc, st)
	if err != nil {
		return "", err
	}
	result["notice"] = "自动交易已启用；系统将在连续竞价时段按分钟评估，当前操作也已记录一轮决策"
	return envelope(result), nil
}

func (s *Service) runAccountOnce(ctx context.Context, acc account, st strategyVersion) (map[string]any, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	traceID := trace.FromContext(ctx)
	if traceID == "" {
		traceID = idgen.NewEventID()
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO auto_trade_runs(auto_account_id,strategy_version_id,user_id,status,started_at,trace_id) VALUES(?,?,?,'RUNNING',CURRENT_TIMESTAMP,?)`, acc.ID, st.ID, acc.UserID, traceID)
	if err != nil {
		return nil, err
	}
	runID, _ := res.LastInsertId()
	summary, runErr := s.executeRun(ctx, acc, st, uint64(runID), traceID)
	statusText := "SUCCESS"
	errText := ""
	if runErr != nil {
		statusText = "FAILED"
		errText = runErr.Error()
	}
	_, finalizeErr := s.db.ExecContext(ctx, `UPDATE auto_trade_runs SET status=?,universe_size=?,evaluated_count=?,order_count=?,finished_at=CURRENT_TIMESTAMP,error_message=? WHERE id=?`, statusText, summary.Universe, summary.Evaluated, summary.Orders, errText, runID)
	_, stateErr := s.db.ExecContext(ctx, `INSERT INTO auto_trade_scheduler_state(auto_account_id,last_intraday_run_at,last_error) VALUES(?,CURRENT_TIMESTAMP,?) ON DUPLICATE KEY UPDATE last_intraday_run_at=CURRENT_TIMESTAMP,last_error=VALUES(last_error)`, acc.ID, errText)
	runErr = errors.Join(runErr, finalizeErr, stateErr)
	if runErr != nil {
		return nil, runErr
	}
	return map[string]any{"run_id": runID, "status": statusText, "universe_size": summary.Universe, "evaluated_count": summary.Evaluated, "order_count": summary.Orders, "trading_session": markettime.IsContinuousAuction(time.Now()), "disclaimer": disclaimer}, nil
}

func (s *Service) Pause(ctx context.Context, userID, accountID uint64) (string, error) {
	if _, err := s.getAccount(ctx, userID, accountID); err != nil {
		return "", err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE auto_trade_accounts SET status='PAUSED' WHERE user_id=? AND id=?`, userID, accountID)
	if err != nil {
		return "", err
	}
	return envelope(map[string]any{"id": accountID, "status": "PAUSED"}), nil
}

func (s *Service) ListRuns(ctx context.Context, userID, accountID uint64) (string, error) {
	if _, err := s.getAccount(ctx, userID, accountID); err != nil {
		return "", err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,strategy_version_id,status,universe_size,evaluated_count,order_count,started_at,finished_at,trace_id,error_message FROM auto_trade_runs WHERE user_id=? AND auto_account_id=? ORDER BY id DESC LIMIT 100`, userID, accountID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, sv uint64
		var statusText, traceID, errText string
		var universe, evaluated, orders int
		var started time.Time
		var finished sql.NullTime
		if err := rows.Scan(&id, &sv, &statusText, &universe, &evaluated, &orders, &started, &finished, &traceID, &errText); err != nil {
			return "", err
		}
		item := map[string]any{"id": id, "strategy_version_id": sv, "status": statusText, "universe_size": universe, "evaluated_count": evaluated, "order_count": orders, "started_at": formatTime(started), "trace_id": traceID, "error_message": errText}
		if finished.Valid {
			item["finished_at"] = formatTime(finished.Time)
		}
		items = append(items, item)
	}
	return envelope(map[string]any{"items": items}), rows.Err()
}

func (s *Service) ListDecisions(ctx context.Context, userID, accountID uint64) (string, error) {
	if _, err := s.getAccount(ctx, userID, accountID); err != nil {
		return "", err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT id,run_id,strategy_version_id,symbol,`signal`,score,action,quantity,limit_price,reason,risk_result,CAST(evidence_json AS CHAR),COALESCE(paper_order_id,0),trace_id,created_at FROM auto_trade_decisions WHERE user_id=? AND auto_account_id=? ORDER BY id DESC LIMIT 300", userID, accountID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, runID, sv, paperID uint64
		var symbol, signal, action, reason, riskResult, evidenceJSON, traceID string
		var score, price float64
		var qty int64
		var created time.Time
		if err := rows.Scan(&id, &runID, &sv, &symbol, &signal, &score, &action, &qty, &price, &reason, &riskResult, &evidenceJSON, &paperID, &traceID, &created); err != nil {
			return "", err
		}
		var evidence any
		_ = json.Unmarshal([]byte(evidenceJSON), &evidence)
		items = append(items, map[string]any{"id": id, "run_id": runID, "strategy_version_id": sv, "symbol": symbol, "signal": signal, "score": score, "action": action, "quantity": qty, "limit_price": price, "reason": reason, "risk_result": riskResult, "evidence": evidence, "paper_order_id": paperID, "trace_id": traceID, "created_at": formatTime(created)})
	}
	return envelope(map[string]any{"items": items, "disclaimer": disclaimer}), rows.Err()
}

func (s *Service) ListDeliveryStatements(ctx context.Context, in *autotrader.DeliveryStatementReq) (string, error) {
	acc, err := s.getAccount(ctx, in.GetUserId(), in.GetAccountId())
	if err != nil {
		return "", err
	}
	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = 500
	}
	if limit > 5000 {
		limit = 5000
	}
	query := `SELECT pt.id,pt.trade_no,pt.order_id,pt.order_no,pt.symbol,COALESCE(pt.name,''),pt.side,
		po.order_type,COALESCE(po.price,0),pt.price,pt.quantity,pt.amount,pt.fee,pt.stamp_duty,pt.slippage,
		pt.depth_level,pt.quote_time,pt.data_mode,pt.trace_id,pt.trade_time,
		d.id,d.strategy_version_id,sv.version_no,sv.name,d.reason
		FROM auto_trade_decisions d
		JOIN paper_orders po ON po.id=d.paper_order_id AND po.user_id=d.user_id
		JOIN paper_trades pt ON pt.order_id=po.id AND pt.user_id=d.user_id
		JOIN auto_strategy_versions sv ON sv.id=d.strategy_version_id AND sv.auto_account_id=d.auto_account_id
		WHERE d.user_id=? AND d.auto_account_id=? AND po.account_id=?`
	args := []any{in.GetUserId(), in.GetAccountId(), acc.PaperAccountID}
	if start := strings.TrimSpace(in.GetStartDate()); start != "" {
		if _, parseErr := time.Parse("2006-01-02", start); parseErr != nil {
			return "", status.Error(codes.InvalidArgument, "开始日期格式应为 YYYY-MM-DD")
		}
		query += " AND pt.trade_time>=?"
		args = append(args, start+" 00:00:00")
	}
	if end := strings.TrimSpace(in.GetEndDate()); end != "" {
		parsed, parseErr := time.Parse("2006-01-02", end)
		if parseErr != nil {
			return "", status.Error(codes.InvalidArgument, "结束日期格式应为 YYYY-MM-DD")
		}
		query += " AND pt.trade_time<?"
		args = append(args, parsed.AddDate(0, 0, 1).Format("2006-01-02 15:04:05"))
	}
	if symbol := strings.TrimSpace(in.GetSymbol()); symbol != "" {
		query += " AND pt.symbol=?"
		args = append(args, symbol)
	}
	query += " ORDER BY pt.trade_time DESC,pt.id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var tradeID, orderID, decisionID, strategyID uint64
		var versionNo, quantity, depthLevel int
		var tradeNo, orderNo, symbol, name, side, orderType, quoteTime, dataMode, traceID, strategyName, reason string
		var orderPrice, fillPrice, amount, fee, stampDuty, slippage float64
		var tradeTime time.Time
		if err := rows.Scan(&tradeID, &tradeNo, &orderID, &orderNo, &symbol, &name, &side, &orderType,
			&orderPrice, &fillPrice, &quantity, &amount, &fee, &stampDuty, &slippage, &depthLevel,
			&quoteTime, &dataMode, &traceID, &tradeTime, &decisionID, &strategyID, &versionNo, &strategyName, &reason); err != nil {
			return "", err
		}
		items = append(items, map[string]any{
			"trade_id": tradeID, "trade_no": tradeNo, "order_id": orderID, "order_no": orderNo,
			"symbol": symbol, "name": name, "side": side, "order_type": orderType,
			"order_price": orderPrice, "fill_price": fillPrice, "quantity": quantity, "amount": amount,
			"fee": fee, "stamp_duty": stampDuty, "slippage": slippage, "depth_level": depthLevel,
			"quote_time": quoteTime, "data_mode": dataMode, "trace_id": traceID, "trade_time": formatTime(tradeTime),
			"decision_id": decisionID, "strategy_version_id": strategyID, "strategy_version": versionNo,
			"strategy_name": strategyName, "decision_reason": reason,
		})
	}
	return envelope(map[string]any{"items": items, "disclaimer": disclaimer}), rows.Err()
}

type runSummary struct{ Universe, Evaluated, Orders int }

func (s *Service) executeRun(ctx context.Context, acc account, st strategyVersion, runID uint64, traceID string) (runSummary, error) {
	strategyCode, paramsJSON, configErr := effectiveStrategyConfig(st)
	if configErr != nil {
		return runSummary{}, status.Errorf(codes.FailedPrecondition, "策略参数无效: %v", configErr)
	}
	analysisLimit := acc.MaxSymbols * 3
	if analysisLimit < 10 {
		analysisLimit = 10
	}
	if analysisLimit > 50 {
		analysisLimit = 50
	}
	reply, err := s.feature.Candidates(ctx, &featureservice.CandidateReq{SortBy: "total", MinConfidence: 0, Limit: analysisLimit, Refresh: false})
	if err != nil {
		return runSummary{}, err
	}
	var payload struct {
		Data struct {
			Items []candidate `json:"items"`
		} `json:"data"`
	}
	if err = json.Unmarshal([]byte(reply.Json), &payload); err != nil {
		return runSummary{}, err
	}
	summary := runSummary{Universe: len(payload.Data.Items)}
	positions, _ := s.paper.ListPositions(ctx, &paperclient.ListPositionsReq{UserId: acc.UserID, AccountId: acc.PaperAccountID})
	positionBySymbol := map[string]*paperclient.Position{}
	for _, p := range positions.GetItems() {
		positionBySymbol[p.GetSymbol()] = p
	}
	assets, err := s.paper.GetAccountAssets(ctx, &paperclient.GetAccountAssetsReq{UserId: acc.UserID, AccountId: acc.PaperAccountID})
	if err != nil {
		return summary, err
	}
	totalAsset := parseFloat(assets.Account.TotalAsset)
	availableCash := parseFloat(assets.Account.Cash)
	dayStartAsset, dayErr := s.ensureDayStartAsset(ctx, acc.ID, totalAsset)
	if dayErr != nil {
		return summary, dayErr
	}
	if dayStartAsset > 0 && (totalAsset-dayStartAsset)/dayStartAsset <= -acc.MaxDailyLossRatio {
		return summary, status.Error(codes.FailedPrecondition, "账户亏损已达到风险预算，自动交易已停止")
	}
	grossExposure := 0.0
	for _, position := range positions.GetItems() {
		grossExposure += parseFloat(position.GetMarketValue())
	}
	for _, candidate := range payload.Data.Items {
		if candidate.DataMode != "real" {
			continue
		}
		summary.Evaluated++
		signalReply, runErr := s.strategy.RunStrategy(ctx, &strategyclient.RunStrategyReq{UserId: acc.UserID, StrategyCode: strategyCode, Symbol: candidate.Symbol, ParamsJson: string(paramsJSON)})
		if runErr != nil || len(signalReply.GetSignals()) == 0 {
			if ctx.Err() != nil {
				return summary, ctx.Err()
			}
			continue
		}
		sig := signalReply.Signals[0]
		riskResult := "UNKNOWN"
		var riskErr error
		action, qty, limitPrice, reason := "HOLD", int64(0), 0.0, sig.Reason
		bookData, bookErr := s.realBook(ctx, candidate.Symbol)
		if sig.Score < acc.MinSignalScore {
			reason = fmt.Sprintf("信号分 %.1f 低于执行阈值 %.1f；%s", sig.Score, acc.MinSignalScore, reason)
		} else {
			riskReply, currentRiskErr := s.risk.AnalyzeStockRisk(ctx, &riskclient.AnalyzeStockRiskReq{UserId: acc.UserID, Symbol: candidate.Symbol})
			riskErr = currentRiskErr
			if riskErr == nil {
				riskResult = riskReply.OverallRiskLevel
			}
		}
		if sig.Score >= acc.MinSignalScore && (riskErr != nil || riskResult == "HIGH" || riskResult == "EXTREME") {
			action = "RISK_REJECTED"
			reason = "风控等级不允许执行；" + reason
		} else if sig.Score >= acc.MinSignalScore && !markettime.IsContinuousAuction(time.Now()) {
			action = "WAIT_MARKET"
			reason = "当前不在 A 股连续竞价时段，仅记录决策；" + reason
		} else if sig.Score >= acc.MinSignalScore && bookErr != nil {
			action = "DATA_REJECTED"
			reason = "真实五档盘口不可用，禁止自动委托；" + reason
		} else if sig.Score >= acc.MinSignalScore && sig.SignalType == "BUY" && len(bookData.Asks) > 0 {
			if _, exists := positionBySymbol[candidate.Symbol]; exists || len(positionBySymbol) >= int(acc.MaxSymbols) {
				reason = "已持有该股票或达到最大持股数；" + reason
				action = "HOLD"
			} else {
				limitPrice = bookData.Asks[0].Price
				remainingGrossBudget := totalAsset*acc.MaxTotalPositionRatio - grossExposure
				budget := math.Min(totalAsset*acc.MaxPositionRatio, math.Min(remainingGrossBudget, availableCash))
				qty = int64(math.Floor(budget/limitPrice/100)) * 100
				if qty >= 100 {
					action = "BUY"
				} else {
					action = "RISK_REJECTED"
					reason = "组合总仓位或可用资金预算不足；" + reason
				}
			}
		} else if sig.Score >= acc.MinSignalScore && sig.SignalType == "SELL" && len(bookData.Bids) > 0 && positionBySymbol[candidate.Symbol] != nil {
			limitPrice = bookData.Bids[0].Price
			qty = positionBySymbol[candidate.Symbol].AvailableVolume
			if qty > 0 {
				action = "SELL"
			}
		}
		var paperOrderID uint64
		if action == "BUY" || action == "SELL" {
			orderReply, orderErr := s.paper.CreateOrder(ctx, &paperclient.CreateOrderReq{UserId: acc.UserID, AccountId: acc.PaperAccountID, ClientOrderId: fmt.Sprintf("auto-%d-%s-%s", runID, candidate.Symbol, action), Symbol: candidate.Symbol, Name: candidate.Name, Side: action, OrderType: "LIMIT", Price: fmt.Sprintf("%.4f", limitPrice), Volume: qty})
			if orderErr != nil {
				action = "ORDER_REJECTED"
				reason = orderErr.Error() + "；" + reason
			} else {
				paperOrderID = orderReply.Order.Id
				summary.Orders++
				if action == "BUY" {
					committed := limitPrice * float64(qty)
					grossExposure += committed
					availableCash -= committed
				}
			}
		}
		evidence := map[string]any{"candidate": candidate, "strategy_signal": map[string]any{"type": sig.SignalType, "score": sig.Score, "reason": sig.Reason, "trigger_price": sig.TriggerPrice, "raw": json.RawMessage(sig.RawDataJson)}, "risk_level": riskResult, "orderbook": bookData}
		evidenceJSON, _ := json.Marshal(evidence)
		_, _ = s.db.ExecContext(ctx, "INSERT INTO auto_trade_decisions(run_id,auto_account_id,strategy_version_id,user_id,symbol,`signal`,score,action,quantity,limit_price,reason,risk_result,evidence_json,paper_order_id,trace_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,CAST(? AS JSON),NULLIF(?,0),?)", runID, acc.ID, st.ID, acc.UserID, candidate.Symbol, sig.SignalType, sig.Score, action, qty, limitPrice, reason, riskResult, string(evidenceJSON), paperOrderID, traceID)
	}
	return summary, nil
}

func (s *Service) realBook(ctx context.Context, symbol string) (book, error) {
	reply, err := s.market.OrderBook(ctx, &marketclient.SymbolRequest{Symbol: symbol})
	if err != nil {
		return book{}, err
	}
	var payload struct {
		Code int  `json:"code"`
		Data book `json:"data"`
	}
	if err = json.Unmarshal([]byte(reply.Json), &payload); err != nil {
		return book{}, err
	}
	if payload.Code != 0 || payload.Data.DataMode != "real" || payload.Data.QuoteTime == "" || len(payload.Data.Bids) == 0 || len(payload.Data.Asks) == 0 {
		return book{}, errors.New("真实盘口不可用")
	}
	if strings.TrimSpace(payload.Data.Symbol) != strings.TrimSpace(symbol) {
		return book{}, errors.New("盘口标的与请求股票不一致")
	}
	if err := markettime.ValidateFreshQuote(time.Now(), payload.Data.QuoteTime, markettime.DefaultMaxQuoteAge); err != nil {
		return book{}, err
	}
	return payload.Data, nil
}

func (s *Service) getAccount(ctx context.Context, userID, accountID uint64) (account, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,user_id,paper_account_id,name,status,initial_capital,max_position_ratio,max_total_position_ratio,max_daily_loss_ratio,min_signal_score,max_symbols,created_at,archived_at FROM auto_trade_accounts WHERE user_id=? AND id=? AND deleted_at IS NULL`, userID, accountID)
	item, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return account{}, status.Error(codes.NotFound, "自动交易账户不存在")
	}
	return item, err
}
func scanAccount(row interface{ Scan(...any) error }) (account, error) {
	var a account
	var created time.Time
	var archived sql.NullTime
	err := row.Scan(&a.ID, &a.UserID, &a.PaperAccountID, &a.Name, &a.Status, &a.InitialCapital, &a.MaxPositionRatio, &a.MaxTotalPositionRatio, &a.MaxDailyLossRatio, &a.MinSignalScore, &a.MaxSymbols, &created, &archived)
	a.CreatedAt = formatTime(created)
	if archived.Valid {
		a.ArchivedAt = formatTime(archived.Time)
	}
	return a, err
}
func (s *Service) getStrategy(ctx context.Context, userID, strategyID uint64) (strategyVersion, error) {
	item, err := scanStrategy(s.db.QueryRowContext(ctx, `SELECT id,auto_account_id,version_no,name,base_strategy,CAST(config_json AS CHAR),source,lifecycle,model_provider,model_name,COALESCE(rationale,''),created_at FROM auto_strategy_versions WHERE user_id=? AND id=?`, userID, strategyID))
	if errors.Is(err, sql.ErrNoRows) {
		return strategyVersion{}, status.Error(codes.NotFound, "策略版本不存在")
	}
	return item, err
}
func (s *Service) promotedStrategy(ctx context.Context, userID, accountID uint64) (strategyVersion, error) {
	return scanStrategy(s.db.QueryRowContext(ctx, `SELECT id,auto_account_id,version_no,name,base_strategy,CAST(config_json AS CHAR),source,lifecycle,model_provider,model_name,COALESCE(rationale,''),created_at FROM auto_strategy_versions WHERE user_id=? AND auto_account_id=? AND lifecycle='PROMOTED' ORDER BY version_no DESC LIMIT 1`, userID, accountID))
}
func scanStrategy(row interface{ Scan(...any) error }) (strategyVersion, error) {
	var st strategyVersion
	var configJSON string
	var created time.Time
	err := row.Scan(&st.ID, &st.AutoAccountID, &st.VersionNo, &st.Name, &st.BaseStrategy, &configJSON, &st.Source, &st.Lifecycle, &st.ModelProvider, &st.ModelName, &st.Rationale, &created)
	if err == nil {
		_ = json.Unmarshal([]byte(configJSON), &st.Config)
		st.CreatedAt = formatTime(created)
	}
	return st, err
}

func (s *Service) ensureDayStartAsset(ctx context.Context, accountID uint64, currentAsset float64) (float64, error) {
	day := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02")
	_, err := s.db.ExecContext(ctx, `INSERT INTO auto_trade_scheduler_state(auto_account_id,risk_day,day_start_asset) VALUES(?,?,?) ON DUPLICATE KEY UPDATE day_start_asset=IF(risk_day=VALUES(risk_day),day_start_asset,VALUES(day_start_asset)),risk_day=VALUES(risk_day)`, accountID, day, currentAsset)
	if err != nil {
		return 0, err
	}
	var start float64
	err = s.db.QueryRowContext(ctx, `SELECT day_start_asset FROM auto_trade_scheduler_state WHERE auto_account_id=?`, accountID).Scan(&start)
	return start, err
}

func parseGeneratedConfig(content string) (map[string]any, bool) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	var out map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(content)), &out) != nil {
		return nil, false
	}
	code, _, err := effectiveStrategyConfig(strategyVersion{BaseStrategy: stringValue(out["base_strategy"], ""), Config: out})
	if err != nil {
		return nil, false
	}
	out["base_strategy"] = code
	return out, true
}
func effectiveStrategyConfig(st strategyVersion) (string, []byte, error) {
	params, ok := st.Config["params"].(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("缺少参数对象")
	}
	code, err := strategyparams.Validate(st.BaseStrategy, params)
	if err != nil {
		return "", nil, err
	}
	encoded, err := json.Marshal(params)
	return code, encoded, err
}
func strategyLabel(v string) string {
	m := map[string]string{"trend_score": "趋势评分", "ma_cross": "均线交叉", "macd_cross": "MACD 交叉", "rsi_reversal": "RSI 反转", "boll_breakout": "布林带突破", "volume_breakout": "放量突破"}
	return m[v]
}
func stringValue(v any, fallback string) string {
	if text, ok := v.(string); ok && strings.TrimSpace(text) != "" {
		return text
	}
	return fallback
}
func clamp(value, min, max, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
func parseFloat(v string) float64 { var out float64; fmt.Sscan(v, &out); return out }
func envelope(data any) string {
	body, _ := json.Marshal(map[string]any{"code": 0, "message": "success", "data": data})
	return string(body)
}
func formatTime(t time.Time) string { return t.Format("2006-01-02 15:04:05") }
