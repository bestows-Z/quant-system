package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"quant-system/backend-go/apps/strategy-rpc/internal/client"
	"quant-system/backend-go/apps/strategy-rpc/strategy"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"
	"quant-system/backend-go/common/xmq"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const Disclaimer = "策略信号仅供研究，不构成投资建议"

type Service struct {
	db       *sql.DB
	quant    *client.QuantPythonClient
	producer xmq.Producer
}

func NewService(db *sql.DB, quant *client.QuantPythonClient, producer xmq.Producer) *Service {
	return &Service{db: db, quant: quant, producer: producer}
}

func (s *Service) EnsureSystemStrategies(ctx context.Context) {
	items := []struct{ name, code, desc string }{
		{"均线金叉死叉策略", "ma_cross", "短期均线与长期均线交叉分析"},
		{"MACD 金叉死叉策略", "macd_cross", "MACD DIF/DEA 交叉分析"},
		{"RSI 超买超卖策略", "rsi_reversal", "RSI 超买超卖分析"},
		{"BOLL 突破策略", "boll_breakout", "布林带突破分析"},
		{"放量突破策略", "volume_breakout", "成交量突破分析"},
		{"趋势评分策略", "trend_score", "多因子趋势评分"},
	}
	for _, item := range items {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO strategies (user_id,name,code,strategy_type,description,params_json,status,is_system,enabled)
			VALUES (NULL,?,?,?,?, '{}','ENABLED',1,1)
			ON DUPLICATE KEY UPDATE name=VALUES(name), strategy_type='BUILTIN', description=VALUES(description), status='ENABLED', is_system=1, enabled=1, deleted_at=NULL`, item.name, item.code, "BUILTIN", item.desc)
	}
}

func (s *Service) ListStrategies(ctx context.Context, userID uint64) ([]*strategy.StrategyInfo, error) {
	s.EnsureSystemStrategies(ctx)
	rows, err := s.db.QueryContext(ctx, `SELECT id, COALESCE(user_id,0), name, code, COALESCE(description,''), COALESCE(CAST(params_json AS CHAR),'{}'), status, is_system, created_at FROM strategies WHERE deleted_at IS NULL AND (is_system=1 OR user_id=?) ORDER BY is_system DESC,id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*strategy.StrategyInfo
	for rows.Next() {
		item, err := scanStrategy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) GetStrategy(ctx context.Context, userID, id uint64) (*strategy.StrategyInfo, error) {
	s.EnsureSystemStrategies(ctx)
	item, err := scanStrategy(s.db.QueryRowContext(ctx, `SELECT id, COALESCE(user_id,0), name, code, COALESCE(description,''), COALESCE(CAST(params_json AS CHAR),'{}'), status, is_system, created_at FROM strategies WHERE id=? AND deleted_at IS NULL AND (is_system=1 OR user_id=?)`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "策略不存在")
	}
	return item, err
}

func (s *Service) CreateStrategy(ctx context.Context, in *strategy.CreateStrategyReq) (*strategy.StrategyInfo, error) {
	if in.GetUserId() == 0 || strings.TrimSpace(in.GetName()) == "" || strings.TrimSpace(in.GetCode()) == "" {
		return nil, status.Error(codes.InvalidArgument, "策略参数错误")
	}
	if _, _, err := effectiveStrategyConfig(&strategy.StrategyInfo{Code: in.GetCode(), ParamsJson: normalizeJSON(in.GetParamsJson())}, ""); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO strategies (user_id,name,code,strategy_type,description,params_json,status,is_system,enabled,created_by) VALUES (?,?,?,?,?,?,'ENABLED',0,1,?)`, in.GetUserId(), in.GetName(), in.GetCode(), "CUSTOM", in.GetDescription(), normalizeJSON(in.GetParamsJson()), in.GetUserId())
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetStrategy(ctx, in.GetUserId(), uint64(id))
}

func (s *Service) UpdateStrategy(ctx context.Context, in *strategy.UpdateStrategyReq) (*strategy.StrategyInfo, error) {
	current, err := s.GetStrategy(ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	if _, _, err = effectiveStrategyConfig(&strategy.StrategyInfo{Code: current.Code, ParamsJson: normalizeJSON(in.GetParamsJson())}, ""); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE strategies SET name=IF(?='',name,?), description=?, params_json=?, status=IF(?='',status,?), enabled=(IF(?='DISABLED',0,1)) WHERE id=? AND user_id=? AND is_system=0`, in.GetName(), in.GetName(), in.GetDescription(), normalizeJSON(in.GetParamsJson()), in.GetStatus(), in.GetStatus(), in.GetStatus(), in.GetId(), in.GetUserId()); err != nil {
		return nil, err
	}
	return s.GetStrategy(ctx, in.GetUserId(), in.GetId())
}

func (s *Service) DeleteStrategies(ctx context.Context, userID uint64, ids []uint64) (uint64, error) {
	if userID == 0 || len(ids) == 0 {
		return 0, status.Error(codes.InvalidArgument, "请选择要删除的个人策略")
	}
	if len(ids) > 100 {
		return 0, status.Error(codes.InvalidArgument, "每次最多删除 100 个个人策略")
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+1)
	args = append(args, userID)
	for _, id := range ids {
		args = append(args, id)
	}
	res, err := s.db.ExecContext(ctx, `UPDATE strategies SET deleted_at=CURRENT_TIMESTAMP,enabled=0,status='DISABLED' WHERE user_id=? AND is_system=0 AND deleted_at IS NULL AND id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	count, _ := res.RowsAffected()
	return uint64(count), nil
}

func (s *Service) RunStrategy(ctx context.Context, in *strategy.RunStrategyReq) (*strategy.RunStrategyResp, error) {
	if in.GetUserId() == 0 || strings.TrimSpace(in.GetSymbol()) == "" {
		return nil, status.Error(codes.InvalidArgument, "运行参数错误")
	}
	st, err := s.strategyForRun(ctx, in)
	if err != nil {
		return nil, err
	}
	runNo := "SR" + idgen.NewEventID()
	res, err := s.db.ExecContext(ctx, `INSERT INTO strategy_runs (run_no,strategy_id,user_id,strategy_code,symbol,status,params_json,started_at) VALUES (?,?,?,?,?,'RUNNING',?,CURRENT_TIMESTAMP)`, runNo, st.Id, in.GetUserId(), st.Code, in.GetSymbol(), normalizeJSON(in.GetParamsJson()))
	if err != nil {
		return nil, err
	}
	runID, _ := res.LastInsertId()
	s.publish(ctx, constants.EventStrategyRunCreated, in.GetUserId(), map[string]any{"run_id": runID})
	pyCode, paramsJSON, configErr := effectiveStrategyConfig(st, in.GetParamsJson())
	if configErr != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE strategy_runs SET status='FAILED', error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`, configErr.Error(), runID)
		return nil, status.Error(codes.InvalidArgument, configErr.Error())
	}
	payload, raw, err := s.quant.StrategySignal(ctx, in.GetSymbol(), pyCode, paramsJSON)
	if err != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE strategy_runs SET status='FAILED', error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`, err.Error(), runID)
		s.publish(ctx, constants.EventStrategyRunFailed, in.GetUserId(), map[string]any{"run_id": runID, "error": err.Error()})
		return nil, err
	}
	rawJSON, _ := json.Marshal(raw)
	sig := mapSignal(uint64(runID), in.GetUserId(), st.Id, st.Code, in.GetSymbol(), payload, string(rawJSON))
	if numericValue(sig.TriggerPrice) <= 0 || strings.ToLower(payload.DataMode) != "real" {
		err := fmt.Errorf("策略结果缺少真实有效的触发价格或行情来源")
		_, _ = s.db.ExecContext(ctx, `UPDATE strategy_runs SET status='FAILED', error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`, err.Error(), runID)
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	sigID, err := s.insertSignal(ctx, in.GetUserId(), sig)
	if err != nil {
		return nil, err
	}
	sig.Id = sigID
	_, _ = s.db.ExecContext(ctx, `UPDATE strategy_runs SET status='SUCCESS', result_summary=?, result_json=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`, sig.Reason, string(rawJSON), runID)
	s.publish(ctx, constants.EventStrategySignalGenerated, in.GetUserId(), map[string]any{"run_id": runID, "symbol": in.GetSymbol(), "signal": sig.SignalType})
	s.publish(ctx, constants.EventStrategyRunCompleted, in.GetUserId(), map[string]any{"run_id": runID})
	return &strategy.RunStrategyResp{RunId: uint64(runID), Signals: []*strategy.StrategySignal{sig}, Summary: sig.Reason, Disclaimer: Disclaimer}, nil
}

func (s *Service) GetSignals(ctx context.Context, userID uint64, strategyCode, signalType string, limit int64) ([]*strategy.StrategySignal, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT id, COALESCE(run_id,0), strategy_id, symbol, COALESCE(name,''), strategy_code, signal_type, COALESCE(signal_strength,''), score, COALESCE(reason,''), risk_level, COALESCE(trigger_price,0), trigger_time, COALESCE(CAST(raw_data_json AS CHAR),CAST(raw_json AS CHAR),'{}'), disclaimer FROM strategy_signals WHERE (user_id=? OR user_id IS NULL)`
	args := []any{userID}
	if strategyCode != "" {
		q += ` AND strategy_code=?`
		args = append(args, strategyCode)
	}
	if signalType != "" {
		q += ` AND signal_type=?`
		args = append(args, signalType)
	}
	q += ` ORDER BY trigger_time DESC LIMIT ?`
	args = append(args, limit)
	return s.querySignals(ctx, q, args...)
}

func (s *Service) GetSignalsBySymbol(ctx context.Context, userID uint64, symbol string, limit int64) ([]*strategy.StrategySignal, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.querySignals(ctx, `SELECT id, COALESCE(run_id,0), strategy_id, symbol, COALESCE(name,''), strategy_code, signal_type, COALESCE(signal_strength,''), score, COALESCE(reason,''), risk_level, COALESCE(trigger_price,0), trigger_time, COALESCE(CAST(raw_data_json AS CHAR),CAST(raw_json AS CHAR),'{}'), disclaimer FROM strategy_signals WHERE symbol=? AND (user_id=? OR user_id IS NULL) ORDER BY trigger_time DESC LIMIT ?`, symbol, userID, limit)
}

func (s *Service) GetLatestSignal(ctx context.Context, userID uint64, symbol string) (*strategy.StrategySignal, error) {
	items, err := s.GetSignalsBySymbol(ctx, userID, symbol, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, status.Error(codes.NotFound, "策略信号不存在")
	}
	return items[0], nil
}

func (s *Service) strategyForRun(ctx context.Context, in *strategy.RunStrategyReq) (*strategy.StrategyInfo, error) {
	if in.GetStrategyId() > 0 {
		return s.GetStrategy(ctx, in.GetUserId(), in.GetStrategyId())
	}
	code := in.GetStrategyCode()
	if code == "" {
		code = "trend_score"
	}
	s.EnsureSystemStrategies(ctx)
	item, err := scanStrategy(s.db.QueryRowContext(ctx, `SELECT id, COALESCE(user_id,0), name, code, COALESCE(description,''), COALESCE(CAST(params_json AS CHAR),'{}'), status, is_system, created_at FROM strategies WHERE code=? AND deleted_at IS NULL AND (is_system=1 OR user_id=?) LIMIT 1`, code, in.GetUserId()))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "策略不存在")
	}
	return item, err
}

func (s *Service) insertSignal(ctx context.Context, userID uint64, sig *strategy.StrategySignal) (uint64, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO strategy_signals (run_id,user_id,strategy_id,strategy_code,symbol,name,`signal`,signal_type,signal_strength,score,reason,risk_level,trigger_price,trigger_time,raw_json,raw_data_json,disclaimer) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,CAST(? AS JSON),CAST(? AS JSON),?)", sig.RunId, userID, sig.StrategyId, sig.StrategyCode, sig.Symbol, sig.Name, sig.SignalType, sig.SignalType, sig.SignalStrength, sig.Score, sig.Reason, sig.RiskLevel, sig.TriggerPrice, sig.TriggerTime, sig.RawDataJson, sig.RawDataJson, sig.Disclaimer)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return uint64(id), nil
}

func (s *Service) querySignals(ctx context.Context, q string, args ...any) ([]*strategy.StrategySignal, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*strategy.StrategySignal
	for rows.Next() {
		var item strategy.StrategySignal
		var price string
		var t time.Time
		if err := rows.Scan(&item.Id, &item.RunId, &item.StrategyId, &item.Symbol, &item.Name, &item.StrategyCode, &item.SignalType, &item.SignalStrength, &item.Score, &item.Reason, &item.RiskLevel, &price, &t, &item.RawDataJson, &item.Disclaimer); err != nil {
			return nil, err
		}
		item.TriggerPrice = price
		item.TriggerTime = t.Format("2006-01-02 15:04:05")
		out = append(out, &item)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanStrategy(s scanner) (*strategy.StrategyInfo, error) {
	var item strategy.StrategyInfo
	var t time.Time
	if err := s.Scan(&item.Id, &item.UserId, &item.Name, &item.Code, &item.Description, &item.ParamsJson, &item.Status, &item.IsSystem, &t); err != nil {
		return nil, err
	}
	item.CreatedAt = t.Format("2006-01-02 15:04:05")
	return &item, nil
}
func normalizeJSON(v string) string {
	if strings.TrimSpace(v) == "" {
		return "{}"
	}
	var x any
	if json.Unmarshal([]byte(v), &x) != nil {
		return "{}"
	}
	return v
}
func pythonStrategyCode(code string) string {
	switch code {
	case "moving_average_cross":
		return "ma_cross"
	case "rsi_overbought_oversold":
		return "rsi_reversal"
	default:
		return code
	}
}
func mapSignal(runID, userID, strategyID uint64, code, symbol string, p *client.StrategySignalPayload, raw string) *strategy.StrategySignal {
	sig := p.Signal
	typ := fmt.Sprint(sig["signal"])
	score := numericValue(sig["score"])
	reason := fmt.Sprint(sig["reason"])
	risk := fmt.Sprint(sig["risk_level"])
	trigger := fmt.Sprint(sig["trigger_time"])
	if trigger == "" || trigger == "<nil>" {
		trigger = time.Now().Format("2006-01-02 15:04:05")
	}
	triggerPrice := fmt.Sprint(sig["trigger_price"])
	if triggerPrice == "" || triggerPrice == "<nil>" {
		triggerPrice = "0"
	}
	return &strategy.StrategySignal{RunId: runID, StrategyId: strategyID, Symbol: symbol, StrategyCode: code, SignalType: typ, SignalStrength: typ, Score: score, Reason: reason, RiskLevel: risk, TriggerPrice: triggerPrice, TriggerTime: strings.ReplaceAll(trigger, "T", " "), RawDataJson: raw, Disclaimer: Disclaimer}
}

func effectiveStrategyConfig(st *strategy.StrategyInfo, runParams string) (string, string, error) {
	base := pythonStrategyCode(st.GetCode())
	params := map[string]any{}
	if err := json.Unmarshal([]byte(normalizeJSON(st.GetParamsJson())), &params); err != nil {
		return "", "", fmt.Errorf("策略保存参数不是有效 JSON")
	}
	if override := strings.TrimSpace(runParams); override != "" && override != "{}" {
		var values map[string]any
		if err := json.Unmarshal([]byte(override), &values); err != nil {
			return "", "", fmt.Errorf("本次运行参数不是有效 JSON")
		}
		for key, value := range values {
			params[key] = value
		}
	}
	if configured, ok := params["base_strategy"].(string); ok && configured != "" {
		base = configured
		delete(params, "base_strategy")
	}
	allowed := map[string]bool{"ma_cross": true, "macd_cross": true, "rsi_reversal": true, "boll_breakout": true, "volume_breakout": true, "trend_score": true}
	if !allowed[base] {
		return "", "", fmt.Errorf("策略算法 %s 不受支持", base)
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return "", "", fmt.Errorf("策略参数序列化失败")
	}
	return base, string(raw), nil
}

func numericValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		n, _ := v.Float64()
		return n
	case string:
		n, _ := strconv.ParseFloat(v, 64)
		return n
	default:
		return 0
	}
}
func (s *Service) publish(ctx context.Context, eventType string, userID uint64, payload map[string]any) {
	if s.producer == nil {
		return
	}
	_ = s.producer.Publish(ctx, xmq.Event{EventID: idgen.NewEventID(), EventType: eventType, TraceID: trace.FromContext(ctx), UserID: userID, Payload: payload, CreatedAt: time.Now()})
}
