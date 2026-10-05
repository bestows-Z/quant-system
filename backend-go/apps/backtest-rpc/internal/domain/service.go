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

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/client"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"
	"quant-system/backend-go/common/xmq"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const Disclaimer = "回测结果仅供研究，不构成投资建议"

type Service struct {
	db       *sql.DB
	quant    *client.QuantPythonClient
	producer xmq.Producer
}

func NewService(db *sql.DB, quant *client.QuantPythonClient, producer xmq.Producer) *Service {
	return &Service{db: db, quant: quant, producer: producer}
}

func (s *Service) RunBacktest(ctx context.Context, in *backtest.RunBacktestReq) (*backtest.RunBacktestResp, error) {
	validated, err := validateRunRequest(in)
	if err != nil {
		return nil, err
	}
	taskNo := "BT" + idgen.NewEventID()
	res, err := s.db.ExecContext(ctx, `INSERT INTO backtest_tasks (task_no,user_id,symbol,strategy_code,start_date,end_date,initial_cash,fee_rate,commission_rate,slippage_rate,status,params_json,started_at) VALUES (?,?,?,?,?,?,?,?,?,?,'RUNNING',?,CURRENT_TIMESTAMP)`, taskNo, in.GetUserId(), validated.Symbol, validated.StrategyCode, validated.StartDate, validated.EndDate, validated.InitialCash, validated.FeeRate, validated.FeeRate, validated.SlippageRate, validated.ParamsJSON)
	if err != nil {
		return nil, err
	}
	taskID, _ := res.LastInsertId()
	s.publish(ctx, constants.EventBacktestTaskCreated, in.GetUserId(), map[string]any{"task_id": taskID})
	payload, raw, err := s.quant.RunBacktest(ctx, validated.PythonRequest)
	if err != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE backtest_tasks SET status='FAILED', error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`, err.Error(), taskID)
		s.publish(ctx, constants.EventBacktestTaskFailed, in.GetUserId(), map[string]any{"task_id": taskID, "error": err.Error()})
		return nil, err
	}
	if err := validateBacktestPayload(payload); err != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE backtest_tasks SET status='FAILED', data_source=?, data_mode=?, error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`, payload.Source, payload.DataMode, err.Error(), taskID)
		s.publish(ctx, constants.EventBacktestTaskFailed, in.GetUserId(), map[string]any{"task_id": taskID, "error": err.Error()})
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	if payload.Symbol != validated.Symbol || payload.Strategy != validated.PythonRequest.Strategy {
		err = fmt.Errorf("回测结果的股票或策略与请求不一致")
		_, _ = s.db.ExecContext(ctx, `UPDATE backtest_tasks SET status='FAILED', error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='RUNNING'`, err.Error(), taskID)
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE backtest_tasks SET status='FAILED', error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=?`, "回测结果序列化失败", taskID)
		return nil, err
	}
	result, trades, curve, orders, err := persistSuccessfulBacktest(ctx, s.db, uint64(taskID), in.GetUserId(), validated.Symbol, validated.StrategyCode, validated.ParamsJSON, payload, string(rawJSON))
	if err != nil {
		_, _ = s.db.ExecContext(ctx, `UPDATE backtest_tasks SET status='FAILED', error_message=?, finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='RUNNING'`, err.Error(), taskID)
		s.publish(ctx, constants.EventBacktestTaskFailed, in.GetUserId(), map[string]any{"task_id": taskID, "error": err.Error()})
		return nil, err
	}
	s.publish(ctx, constants.EventBacktestTaskCompleted, in.GetUserId(), map[string]any{"task_id": taskID})
	return &backtest.RunBacktestResp{TaskId: uint64(taskID), TaskNo: taskNo, Status: "SUCCESS", Result: result, Trades: trades, EquityCurve: curve, Disclaimer: Disclaimer, Orders: orders}, nil
}

func (s *Service) ListTasks(ctx context.Context, userID uint64, statusText string, limit int64) ([]*backtest.BacktestTask, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := `SELECT id,task_no,user_id,symbol,strategy_code,start_date,end_date,initial_cash,fee_rate,slippage_rate,status,COALESCE(error_message,''),created_at,COALESCE(data_source,''),COALESCE(data_mode,''),COALESCE(CAST(params_json AS CHAR),'{}'),COALESCE(adjust_mode,''),COALESCE(engine_version,''),COALESCE(execution_model,''),COALESCE(quality_status,''),COALESCE(CAST(config_json AS CHAR),'{}') FROM backtest_tasks WHERE user_id=?`
	args := []any{userID}
	if statusText != "" {
		q += ` AND status=?`
		args = append(args, statusText)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*backtest.BacktestTask
	for rows.Next() {
		it, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
func (s *Service) GetTask(ctx context.Context, userID, id uint64) (*backtest.BacktestTask, error) {
	it, err := scanTask(s.db.QueryRowContext(ctx, `SELECT id,task_no,user_id,symbol,strategy_code,start_date,end_date,initial_cash,fee_rate,slippage_rate,status,COALESCE(error_message,''),created_at,COALESCE(data_source,''),COALESCE(data_mode,''),COALESCE(CAST(params_json AS CHAR),'{}'),COALESCE(adjust_mode,''),COALESCE(engine_version,''),COALESCE(execution_model,''),COALESCE(quality_status,''),COALESCE(CAST(config_json AS CHAR),'{}') FROM backtest_tasks WHERE user_id=? AND id=?`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "回测任务不存在")
	}
	return it, err
}
func (s *Service) GetResult(ctx context.Context, userID, taskID uint64) (*backtest.BacktestResult, error) {
	it, err := scanResult(s.db.QueryRowContext(ctx, `SELECT r.id,r.task_id,COALESCE(r.symbol,''),COALESCE(r.strategy_code,''),r.total_return,COALESCE(r.annual_return,0),COALESCE(r.max_drawdown,0),COALESCE(r.win_rate,0),COALESCE(r.profit_loss_ratio,0),r.total_trades,r.winning_trades,r.losing_trades,r.final_asset,COALESCE(CAST(r.raw_result_json AS CHAR),CAST(r.result_json AS CHAR),'{}'),r.disclaimer,COALESCE(t.data_source,''),COALESCE(t.data_mode,''),COALESCE(CAST(t.params_json AS CHAR),'{}'),COALESCE(t.engine_version,''),COALESCE(t.execution_model,''),COALESCE(t.adjust_mode,''),COALESCE(t.quality_status,''),COALESCE(CAST(t.config_json AS CHAR),'{}'),COALESCE(CAST(r.quality_json AS CHAR),'{}'),COALESCE(r.realized_pnl,0),COALESCE(r.unrealized_pnl,0),COALESCE(r.total_fees,0),COALESCE(r.turnover_rate,0),COALESCE(r.exposure_rate,0) FROM backtest_results r JOIN backtest_tasks t ON t.id=r.task_id WHERE t.user_id=? AND r.task_id=?`, userID, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "回测结果不存在")
	}
	return it, err
}
func (s *Service) GetTrades(ctx context.Context, userID, taskID uint64) ([]*backtest.BacktestTrade, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tr.id,tr.task_id,tr.symbol,tr.trade_time,tr.side,tr.price,tr.quantity,tr.amount,tr.fee,tr.profit,COALESCE(tr.reason,''),COALESCE(DATE_FORMAT(tr.signal_date,'%Y-%m-%d'),''),COALESCE(tr.base_price,0),COALESCE(tr.commission,0),COALESCE(tr.stamp_duty,0),COALESCE(tr.transfer_fee,0) FROM backtest_trades tr JOIN backtest_tasks t ON t.id=tr.task_id WHERE t.user_id=? AND tr.task_id=? ORDER BY tr.trade_time`, userID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*backtest.BacktestTrade
	for rows.Next() {
		var it backtest.BacktestTrade
		var t time.Time
		var price, amount, fee, profit, basePrice, commission, stampDuty, transferFee string
		if err := rows.Scan(&it.Id, &it.TaskId, &it.Symbol, &t, &it.Side, &price, &it.Volume, &amount, &fee, &profit, &it.Reason, &it.SignalDate, &basePrice, &commission, &stampDuty, &transferFee); err != nil {
			return nil, err
		}
		it.TradeTime = t.Format("2006-01-02")
		it.Price = price
		it.Amount = amount
		it.Fee = fee
		it.Profit = profit
		it.BasePrice = basePrice
		it.Commission = commission
		it.StampDuty = stampDuty
		it.TransferFee = transferFee
		out = append(out, &it)
	}
	return out, rows.Err()
}
func (s *Service) GetCurve(ctx context.Context, userID, taskID uint64) ([]*backtest.EquityPoint, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ec.id,ec.task_id,ec.trade_date,ec.total_asset,ec.cash,ec.position_value,COALESCE(ec.drawdown,0) FROM backtest_equity_curve ec JOIN backtest_tasks t ON t.id=ec.task_id WHERE t.user_id=? AND ec.task_id=? ORDER BY ec.trade_date`, userID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*backtest.EquityPoint
	for rows.Next() {
		var it backtest.EquityPoint
		var d time.Time
		var total, cash, mv string
		if err := rows.Scan(&it.Id, &it.TaskId, &d, &total, &cash, &mv, &it.Drawdown); err != nil {
			return nil, err
		}
		it.TradeDate = d.Format("2006-01-02")
		it.TotalAsset = total
		it.Cash = cash
		it.MarketValue = mv
		out = append(out, &it)
	}
	return out, rows.Err()
}

func (s *Service) GetOrders(ctx context.Context, userID, taskID uint64) ([]*backtest.BacktestOrder, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT o.id,o.task_id,o.symbol,DATE_FORMAT(o.signal_date,'%Y-%m-%d'),COALESCE(DATE_FORMAT(o.trade_date,'%Y-%m-%d'),''),o.side,o.requested_quantity,o.filled_quantity,o.cancelled_quantity,COALESCE(o.base_price,0),COALESCE(o.fill_price,0),o.status,COALESCE(o.reason_code,''),COALESCE(o.reason,''),COALESCE(o.signal_reason,'') FROM backtest_orders o JOIN backtest_tasks t ON t.id=o.task_id WHERE t.user_id=? AND o.task_id=? ORDER BY o.signal_date,o.id`, userID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*backtest.BacktestOrder
	for rows.Next() {
		var item backtest.BacktestOrder
		if err := rows.Scan(&item.Id, &item.TaskId, &item.Symbol, &item.SignalDate, &item.TradeDate, &item.Side,
			&item.RequestedQuantity, &item.FilledQuantity, &item.CancelledQuantity, &item.BasePrice, &item.FillPrice,
			&item.Status, &item.ReasonCode, &item.Reason, &item.SignalReason); err != nil {
			return nil, err
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}

func (s *Service) DeleteTasks(ctx context.Context, userID uint64, taskIDs []uint64, scope string) (int64, bool, error) {
	if userID == 0 {
		return 0, false, status.Error(codes.InvalidArgument, "用户参数错误")
	}
	ids := uniqueIDs(taskIDs)
	if len(ids) == 0 && scope != "failed" {
		return 0, false, status.Error(codes.InvalidArgument, "请选择要删除的回测记录")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()
	query := `SELECT id,status FROM backtest_tasks WHERE user_id=?`
	args := []any{userID}
	if len(ids) > 0 {
		query += ` AND id IN (` + sqlPlaceholders(len(ids)) + `)`
		for _, id := range ids {
			args = append(args, id)
		}
	} else {
		query += ` AND status='FAILED'`
	}
	query += ` FOR UPDATE`
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, false, err
	}
	owned := make([]uint64, 0, len(ids))
	for rows.Next() {
		var id uint64
		var state string
		if err := rows.Scan(&id, &state); err != nil {
			rows.Close()
			return 0, false, err
		}
		if state == "RUNNING" || state == "PENDING" {
			rows.Close()
			return 0, false, status.Error(codes.FailedPrecondition, "运行中的回测不能删除")
		}
		owned = append(owned, id)
	}
	if err := rows.Close(); err != nil {
		return 0, false, err
	}
	if len(owned) == 0 {
		if err := tx.Commit(); err != nil {
			return 0, false, err
		}
		return 0, true, nil
	}
	marks := sqlPlaceholders(len(owned))
	deleteArgs := make([]any, 0, len(owned))
	for _, id := range owned {
		deleteArgs = append(deleteArgs, id)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM risk_records WHERE source='backtest-rpc' AND source_id IN (`+marks+`)`, deleteArgs...); err != nil {
		return 0, false, err
	}
	for _, table := range []string{"backtest_orders", "backtest_equity_curve", "backtest_trades", "backtest_results", "backtest_tasks"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE `+map[bool]string{true: "id", false: "task_id"}[table == "backtest_tasks"]+` IN (`+marks+`)`, deleteArgs...); err != nil {
			return 0, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return int64(len(owned)), false, nil
}

type scanner interface{ Scan(dest ...any) error }

func scanTask(s scanner) (*backtest.BacktestTask, error) {
	var it backtest.BacktestTask
	var sd, ed, created time.Time
	if err := s.Scan(&it.Id, &it.TaskNo, &it.UserId, &it.Symbol, &it.StrategyCode, &sd, &ed, &it.InitialCash, &it.FeeRate, &it.SlippageRate, &it.Status, &it.ErrorMessage, &created, &it.DataSource, &it.DataMode, &it.ParamsJson, &it.Adjust, &it.EngineVersion, &it.ExecutionModel, &it.QualityStatus, &it.ConfigJson); err != nil {
		return nil, err
	}
	it.StartDate = sd.Format("2006-01-02")
	it.EndDate = ed.Format("2006-01-02")
	it.CreatedAt = created.Format("2006-01-02 15:04:05")
	return &it, nil
}
func scanResult(s scanner) (*backtest.BacktestResult, error) {
	var it backtest.BacktestResult
	if err := s.Scan(&it.Id, &it.TaskId, &it.Symbol, &it.StrategyCode, &it.TotalReturn, &it.AnnualReturn, &it.MaxDrawdown, &it.WinRate, &it.ProfitLossRatio, &it.TotalTrades, &it.WinningTrades, &it.LosingTrades, &it.FinalAsset, &it.RawResultJson, &it.Disclaimer, &it.DataSource, &it.DataMode, &it.ParamsJson, &it.EngineVersion, &it.ExecutionModel, &it.Adjust, &it.QualityStatus, &it.ConfigJson, &it.QualityJson, &it.RealizedPnl, &it.UnrealizedPnl, &it.TotalFees, &it.TurnoverRate, &it.ExposureRate); err != nil {
		return nil, err
	}
	return &it, nil
}
func mapResult(taskID uint64, symbol, code, params string, p *client.BacktestPayload, raw string) *backtest.BacktestResult {
	sum := p.Summary
	total := len(p.Trades)
	configJSON, _ := json.Marshal(p.Config)
	qualityJSON, _ := json.Marshal(p.DataQuality)
	return &backtest.BacktestResult{
		TaskId: taskID, Symbol: symbol, StrategyCode: code, TotalReturn: num(sum["total_return"]),
		AnnualReturn: num(sum["annual_return"]), MaxDrawdown: num(sum["max_drawdown"]), WinRate: num(sum["win_rate"]),
		ProfitLossRatio: num(sum["profit_loss_ratio"]), TotalTrades: int64(total), WinningTrades: int64(num(sum["winning_trades"])),
		LosingTrades: int64(num(sum["losing_trades"])), FinalAsset: fmt.Sprint(sum["final_equity"]), RawResultJson: raw,
		Disclaimer: defaultString(p.Disclaimer, Disclaimer), DataSource: p.Source, DataMode: p.DataMode, ParamsJson: params,
		EngineVersion: p.EngineVersion, ExecutionModel: p.ExecutionModel, Adjust: p.Adjust, QualityStatus: p.DataQuality.Status,
		ConfigJson: string(configJSON), QualityJson: string(qualityJSON), RealizedPnl: num(sum["realized_pnl"]),
		UnrealizedPnl: num(sum["unrealized_pnl"]), TotalFees: num(sum["total_fees"]),
		TurnoverRate: num(sum["turnover_rate"]), ExposureRate: num(sum["exposure_rate"]),
	}
}

func validateRealBacktest(source, mode string) error {
	cleanSource := strings.TrimSpace(strings.ToLower(source))
	if strings.ToLower(strings.TrimSpace(mode)) != "real" || cleanSource == "" {
		return errors.New("回测未获得可追溯的真实历史行情，结果不予保存")
	}
	for _, forbidden := range []string{"mock", "bootstrap", "fixture", "synthetic"} {
		if strings.Contains(cleanSource, forbidden) {
			return errors.New("回测数据来源不是可用的真实历史行情，结果不予保存")
		}
	}
	return nil
}

func uniqueIDs(values []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(values))
	out := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func sqlPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}
func parseFloat(v string, fallback float64) float64 {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
func num(v any) float64 { return parseFloat(fmt.Sprint(v), 0) }
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
func defaultString(v, f string) string {
	if strings.TrimSpace(v) == "" {
		return f
	}
	return v
}
func defaultDate(v, f string) string {
	if strings.TrimSpace(v) == "" {
		return f
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
func (s *Service) publish(ctx context.Context, eventType string, userID uint64, payload map[string]any) {
	if s.producer == nil {
		return
	}
	_ = s.producer.Publish(ctx, xmq.Event{EventID: idgen.NewEventID(), EventType: eventType, TraceID: trace.FromContext(ctx), UserID: userID, Payload: payload, CreatedAt: time.Now()})
}
