package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/client"
)

func persistSuccessfulBacktest(
	ctx context.Context,
	db *sql.DB,
	taskID, userID uint64,
	symbol, strategyCode, paramsJSON string,
	payload *client.BacktestPayload,
	rawJSON string,
) (*backtest.BacktestResult, []*backtest.BacktestTrade, []*backtest.EquityPoint, []*backtest.BacktestOrder, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer tx.Rollback()

	result := mapResult(taskID, symbol, strategyCode, paramsJSON, payload, rawJSON)
	qualityJSON, err := json.Marshal(payload.DataQuality)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("serialize data quality: %w", err)
	}
	configJSON, err := json.Marshal(payload.Config)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("serialize backtest config: %w", err)
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO backtest_results
(task_id,user_id,symbol,strategy_code,final_equity,final_asset,total_return,annual_return,max_drawdown,win_rate,profit_loss_ratio,realized_pnl,unrealized_pnl,total_fees,turnover_rate,exposure_rate,quality_json,trade_count,total_trades,winning_trades,losing_trades,result_json,raw_result_json,disclaimer)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,CAST(? AS JSON),?,?,?,?,CAST(? AS JSON),CAST(? AS JSON),?)`,
		result.TaskId, userID, result.Symbol, result.StrategyCode, result.FinalAsset, result.FinalAsset,
		result.TotalReturn, result.AnnualReturn, result.MaxDrawdown, result.WinRate, result.ProfitLossRatio,
		num(payload.Summary["realized_pnl"]), num(payload.Summary["unrealized_pnl"]), num(payload.Summary["total_fees"]),
		num(payload.Summary["turnover_rate"]), num(payload.Summary["exposure_rate"]), string(qualityJSON),
		result.TotalTrades, result.TotalTrades, result.WinningTrades, result.LosingTrades, rawJSON, rawJSON, result.Disclaimer)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("insert backtest result: %w", err)
	}
	resultID, err := res.LastInsertId()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("read backtest result id: %w", err)
	}
	result.Id = uint64(resultID)

	orders := make([]*backtest.BacktestOrder, 0, len(payload.Orders))
	for index, item := range payload.Orders {
		order := &backtest.BacktestOrder{
			TaskId: taskID, Symbol: symbol, SignalDate: stringValue(item["signal_date"]), TradeDate: optionalStringValue(item["trade_date"]),
			Side: stringValue(item["side"]), RequestedQuantity: integerValue(item["requested_quantity"]),
			FilledQuantity: integerValue(item["filled_quantity"]), CancelledQuantity: integerValue(item["cancelled_quantity"]),
			BasePrice: optionalStringValue(item["base_price"]), FillPrice: optionalStringValue(item["fill_price"]),
			Status: stringValue(item["status"]), ReasonCode: stringValue(item["reason_code"]),
			Reason: stringValue(item["reason"]), SignalReason: stringValue(item["signal_reason"]),
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO backtest_orders
(task_id,user_id,symbol,signal_date,trade_date,side,requested_quantity,filled_quantity,cancelled_quantity,base_price,fill_price,limit_up,limit_down,status,reason_code,reason,signal_reason)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			taskID, userID, symbol, stringValue(item["signal_date"]), nullableString(item["trade_date"]), stringValue(item["side"]),
			integerValue(item["requested_quantity"]), integerValue(item["filled_quantity"]), integerValue(item["cancelled_quantity"]),
			nullableNumber(item["base_price"]), nullableNumber(item["fill_price"]), nullableNumber(item["limit_up"]), nullableNumber(item["limit_down"]),
			stringValue(item["status"]), stringValue(item["reason_code"]), stringValue(item["reason"]), stringValue(item["signal_reason"]))
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("insert backtest order %d: %w", index+1, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("read backtest order id %d: %w", index+1, err)
		}
		order.Id = uint64(id)
		orders = append(orders, order)
	}

	trades := make([]*backtest.BacktestTrade, 0, len(payload.Trades))
	for index, item := range payload.Trades {
		trade := &backtest.BacktestTrade{
			TaskId: taskID, Symbol: symbol, TradeTime: stringValue(item["trade_date"]), Side: stringValue(item["side"]),
			Price: stringValue(item["price"]), Volume: integerValue(item["quantity"]), Amount: stringValue(item["amount"]),
			Fee: stringValue(item["fee"]), Profit: stringValue(item["profit"]), Reason: stringValue(item["reason"]),
			SignalDate: stringValue(item["signal_date"]), BasePrice: stringValue(item["base_price"]),
			Commission: stringValue(item["commission"]), StampDuty: stringValue(item["stamp_duty"]),
			TransferFee: stringValue(item["transfer_fee"]),
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO backtest_trades
(task_id,user_id,symbol,signal_date,trade_time,side,base_price,price,quantity,amount,fee,commission,stamp_duty,transfer_fee,profit,reason)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			taskID, userID, symbol, stringValue(item["signal_date"]), trade.TradeTime, trade.Side, nullableNumber(item["base_price"]),
			trade.Price, trade.Volume, trade.Amount, trade.Fee, numberOrZero(item["commission"]), numberOrZero(item["stamp_duty"]),
			numberOrZero(item["transfer_fee"]), trade.Profit, trade.Reason)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("insert backtest trade %d: %w", index+1, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("read backtest trade id %d: %w", index+1, err)
		}
		trade.Id = uint64(id)
		trades = append(trades, trade)
	}

	curve := make([]*backtest.EquityPoint, 0, len(payload.EquityCurve))
	for index, item := range payload.EquityCurve {
		point := &backtest.EquityPoint{
			TaskId: taskID, TradeDate: stringValue(item["trade_date"]), TotalAsset: stringValue(item["equity"]),
			Cash: stringValue(item["cash"]), MarketValue: stringValue(item["position_value"]), Drawdown: num(item["drawdown"]),
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO backtest_equity_curve
(task_id,user_id,trade_date,equity,total_asset,cash,position_value,drawdown) VALUES (?,?,?,?,?,?,?,?)`,
			taskID, userID, point.TradeDate, point.TotalAsset, point.TotalAsset, point.Cash, point.MarketValue, point.Drawdown)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("insert backtest equity point %d: %w", index+1, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("read backtest equity id %d: %w", index+1, err)
		}
		point.Id = uint64(id)
		curve = append(curve, point)
	}

	update, err := tx.ExecContext(ctx, `UPDATE backtest_tasks SET data_source=?,data_mode=?,adjust_mode=?,engine_version=?,execution_model=?,quality_status=?,config_json=CAST(? AS JSON),status='SUCCESS',error_message=NULL,finished_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=? AND status='RUNNING'`,
		payload.Source, payload.DataMode, payload.Adjust, payload.EngineVersion, payload.ExecutionModel, payload.DataQuality.Status, string(configJSON), taskID, userID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("complete backtest task: %w", err)
	}
	affected, err := update.RowsAffected()
	if err != nil || affected != 1 {
		return nil, nil, nil, nil, fmt.Errorf("complete backtest task: expected one running task, affected=%d", affected)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, nil, nil, err
	}
	return result, trades, curve, orders, nil
}

func stringValue(value any) string { return fmt.Sprint(value) }

func optionalStringValue(value any) string {
	if value == nil {
		return ""
	}
	return stringValue(value)
}

func integerValue(value any) int64 { return int64(num(value)) }

func nullableString(value any) any {
	text := stringValue(value)
	if text == "" || text == "<nil>" {
		return nil
	}
	return text
}

func nullableNumber(value any) any {
	if value == nil {
		return nil
	}
	if number, ok := finiteNumber(value); ok {
		return number
	}
	return nil
}

func numberOrZero(value any) float64 {
	if number, ok := finiteNumber(value); ok {
		return number
	}
	return 0
}
