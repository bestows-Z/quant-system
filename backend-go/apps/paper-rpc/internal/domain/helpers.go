package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/paper-rpc/paper"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"
	"quant-system/backend-go/common/xmq"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type scanner interface{ Scan(dest ...any) error }

func (s *TradingService) createPendingOrder(ctx context.Context, tx *sql.Tx, acc *accountRow, orderNo, name string, price, amount, fee decimal.Decimal, in *paper.CreateOrderReq) (*paper.Order, error) {
	reservedCash := decimal.Zero
	if in.GetSide() == "BUY" {
		frozen := amount.Add(fee)
		reservedCash = frozen
		if acc.Cash.LessThan(frozen) {
			return nil, status.Error(codes.FailedPrecondition, "资金不足")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash=available_cash-?, frozen_cash=frozen_cash+? WHERE id=?`, frozen, frozen, acc.ID); err != nil {
			return nil, err
		}
	} else {
		if err := s.freezeSellPosition(ctx, tx, in.GetUserId(), in.GetAccountId(), in.GetSymbol(), in.GetVolume()); err != nil {
			return nil, err
		}
	}
	id, err := s.insertOrder(ctx, tx, orderNo, name, "PENDING", price, decimal.Zero, decimal.Zero, decimal.Zero, "限价单未触发，已进入 PENDING", in)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE paper_orders SET remaining_quantity=quantity, reserved_cash=?, expire_at=TIMESTAMP(CURRENT_DATE, '15:00:00'), data_mode='real', trace_id=? WHERE id=?`, reservedCash, trace.FromContext(ctx), id); err != nil {
		return nil, err
	}
	if err := s.refreshAccountTx(ctx, tx, in.GetAccountId()); err != nil {
		return nil, err
	}
	return s.orderPBByID(ctx, tx, in.GetUserId(), uint64(id))
}

func (s *TradingService) fillOrder(ctx context.Context, tx *sql.Tx, acc *accountRow, orderNo, name string, price, amount, fee, stampDuty, slippage decimal.Decimal, in *paper.CreateOrderReq) (*paper.Order, error) {
	if in.GetSide() == "BUY" {
		total := amount.Add(fee)
		if acc.Cash.LessThan(total) {
			return nil, status.Error(codes.FailedPrecondition, "资金不足")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash=available_cash-? WHERE id=?`, total, acc.ID); err != nil {
			return nil, err
		}
		if err := s.addBuyPosition(ctx, tx, in.GetUserId(), in.GetAccountId(), in.GetSymbol(), name, in.GetVolume(), price, amount); err != nil {
			return nil, err
		}
	} else {
		if err := s.reduceSellPosition(ctx, tx, in.GetUserId(), in.GetAccountId(), in.GetSymbol(), in.GetVolume(), price); err != nil {
			return nil, err
		}
		receive := amount.Sub(fee).Sub(stampDuty)
		if _, err := tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash=available_cash+? WHERE id=?`, receive, acc.ID); err != nil {
			return nil, err
		}
	}
	id, err := s.insertOrder(ctx, tx, orderNo, name, "FILLED", price, amount, fee, slippage, "立即成交，paper trading / 模拟交易", in)
	if err != nil {
		return nil, err
	}
	tradeNo := "PT" + idgen.NewEventID()
	if _, err = tx.ExecContext(ctx, `
INSERT INTO paper_trades (trade_no, account_id, order_id, order_no, user_id, symbol, name, side, price, quantity, amount, fee, stamp_duty, slippage, trade_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		tradeNo, in.GetAccountId(), id, orderNo, in.GetUserId(), in.GetSymbol(), name, in.GetSide(), price, in.GetVolume(), amount, fee, stampDuty, slippage); err != nil {
		return nil, err
	}
	if err := s.refreshAccountTx(ctx, tx, in.GetAccountId()); err != nil {
		return nil, err
	}
	return s.orderPBByID(ctx, tx, in.GetUserId(), uint64(id))
}

func (s *TradingService) insertOrder(ctx context.Context, tx *sql.Tx, orderNo, name, statusText string, price, amount, fee, slippage decimal.Decimal, remark string, in *paper.CreateOrderReq) (int64, error) {
	res, err := tx.ExecContext(ctx, `
INSERT INTO paper_orders (order_no, client_order_id, account_id, user_id, symbol, name, side, order_type, price, quantity, filled_quantity, avg_fill_price, status, amount, fee, slippage, submitted_at, filled_at, remark)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CASE WHEN ?='FILLED' THEN CURRENT_TIMESTAMP ELSE NULL END, ?)`,
		orderNo, in.GetClientOrderId(), in.GetAccountId(), in.GetUserId(), in.GetSymbol(), name, in.GetSide(), in.GetOrderType(), price, in.GetVolume(), filledVolume(statusText, in.GetVolume()), avgPrice(statusText, price), statusText, amount, fee, slippage, statusText, remark)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *TradingService) addBuyPosition(ctx context.Context, tx *sql.Tx, userID, accountID uint64, symbol, name string, volume int64, price, amount decimal.Decimal) error {
	var oldQty int64
	var oldAvg string
	err := tx.QueryRowContext(ctx, `SELECT quantity, avg_cost FROM paper_positions WHERE account_id=? AND symbol=? FOR UPDATE`, accountID, symbol).Scan(&oldQty, &oldAvg)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, `
INSERT INTO paper_positions (account_id, user_id, symbol, name, quantity, available_quantity, frozen_quantity, avg_cost, market_price, market_value, unrealized_pnl, realized_pnl)
VALUES (?, ?, ?, ?, ?, 0, 0, ?, ?, ?, 0, 0)`, accountID, userID, symbol, name, volume, price, price, amount)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE paper_positions SET today_buy_quantity=?, last_buy_date=CURRENT_DATE WHERE account_id=? AND symbol=?`, volume, accountID, symbol)
		}
		return err
	}
	if err != nil {
		return err
	}
	oldAvgDec := mustDecimal(oldAvg, "0")
	oldAmount := oldAvgDec.Mul(decimal.NewFromInt(oldQty))
	newQty := oldQty + volume
	newAvg := oldAmount.Add(amount).Div(decimal.NewFromInt(newQty))
	newMV := price.Mul(decimal.NewFromInt(newQty))
	_, err = tx.ExecContext(ctx, `
UPDATE paper_positions SET name=?, quantity=?, today_buy_quantity=today_buy_quantity+?, last_buy_date=CURRENT_DATE, avg_cost=?, market_price=?, market_value=?, updated_at=CURRENT_TIMESTAMP
WHERE account_id=? AND symbol=?`, name, newQty, volume, newAvg, price, newMV, accountID, symbol)
	return err
}

func (s *TradingService) reduceSellPosition(ctx context.Context, tx *sql.Tx, userID, accountID uint64, symbol string, volume int64, price decimal.Decimal) error {
	var qty, avail, todayBuy int64
	err := tx.QueryRowContext(ctx, `SELECT quantity, available_quantity, today_buy_quantity FROM paper_positions WHERE user_id=? AND account_id=? AND symbol=? FOR UPDATE`, userID, accountID, symbol).Scan(&qty, &avail, &todayBuy)
	if errors.Is(err, sql.ErrNoRows) {
		return status.Error(codes.FailedPrecondition, "持仓不足")
	}
	if err != nil {
		return err
	}
	if avail < volume || qty < volume {
		if todayBuy > 0 {
			return status.Error(codes.FailedPrecondition, "可用持仓不足（A股 T+1：当日买入不可卖出）")
		}
		return status.Error(codes.FailedPrecondition, "可用持仓不足")
	}
	newQty := qty - volume
	mv := price.Mul(decimal.NewFromInt(newQty))
	_, err = tx.ExecContext(ctx, `UPDATE paper_positions SET quantity=?, available_quantity=available_quantity-?, market_price=?, market_value=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND account_id=? AND symbol=?`,
		newQty, volume, price, mv, userID, accountID, symbol)
	return err
}

func (s *TradingService) freezeSellPosition(ctx context.Context, tx *sql.Tx, userID, accountID uint64, symbol string, volume int64) error {
	var avail, todayBuy int64
	err := tx.QueryRowContext(ctx, `SELECT available_quantity, today_buy_quantity FROM paper_positions WHERE user_id=? AND account_id=? AND symbol=? FOR UPDATE`, userID, accountID, symbol).Scan(&avail, &todayBuy)
	if errors.Is(err, sql.ErrNoRows) {
		return status.Error(codes.FailedPrecondition, "可用持仓不足")
	}
	if err != nil {
		return err
	}
	if avail < volume {
		if todayBuy > 0 {
			return status.Error(codes.FailedPrecondition, "可用持仓不足（A股 T+1：当日买入不可卖出）")
		}
		return status.Error(codes.FailedPrecondition, "可用持仓不足")
	}
	_, err = tx.ExecContext(ctx, `UPDATE paper_positions SET available_quantity=available_quantity-?, frozen_quantity=frozen_quantity+? WHERE user_id=? AND account_id=? AND symbol=?`, volume, volume, userID, accountID, symbol)
	return err
}

func (s *TradingService) selectAccount(ctx context.Context, q queryer, userID, accountID uint64, lock bool) (*accountRow, error) {
	sqlText := `SELECT id, user_id, name, initial_cash, available_cash, frozen_cash, total_assets, total_profit, status, created_at FROM paper_accounts WHERE user_id=? AND id=? AND deleted_at IS NULL`
	if lock {
		sqlText += ` FOR UPDATE`
	}
	row := q.QueryRowContext(ctx, sqlText, userID, accountID)
	acc, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "账户不存在")
	}
	return acc, err
}

func (s *TradingService) selectOrder(ctx context.Context, q queryer, userID, orderID uint64, lock bool) (*orderRow, error) {
	sqlText := `SELECT id, order_no, client_order_id, account_id, user_id, symbol, COALESCE(name,''), side, order_type, COALESCE(price,0), quantity, filled_quantity, remaining_quantity, reserved_cash, COALESCE(avg_fill_price,0), status, amount, fee, COALESCE(remark,''), expire_at, quote_time, data_mode, created_at FROM paper_orders WHERE user_id=? AND id=?`
	if lock {
		sqlText += ` FOR UPDATE`
	}
	o, err := scanOrder(q.QueryRowContext(ctx, sqlText, userID, orderID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "订单不存在")
	}
	return o, err
}

func (s *TradingService) orderByClientID(ctx context.Context, userID uint64, clientOrderID string) (*paper.Order, error) {
	o, err := scanOrder(s.db.QueryRowContext(ctx, `SELECT id, order_no, client_order_id, account_id, user_id, symbol, COALESCE(name,''), side, order_type, COALESCE(price,0), quantity, filled_quantity, remaining_quantity, reserved_cash, COALESCE(avg_fill_price,0), status, amount, fee, COALESCE(remark,''), expire_at, quote_time, data_mode, created_at FROM paper_orders WHERE user_id=? AND client_order_id=?`, userID, clientOrderID))
	if err != nil {
		return nil, err
	}
	return orderPB(o), nil
}

func (s *TradingService) orderPBByID(ctx context.Context, q queryer, userID, orderID uint64) (*paper.Order, error) {
	o, err := s.selectOrder(ctx, q, userID, orderID, false)
	if err != nil {
		return nil, err
	}
	return orderPB(o), nil
}

func (s *TradingService) refreshAccountTx(ctx context.Context, tx *sql.Tx, accountID uint64) error {
	var mvStr sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(market_value),0) FROM paper_positions WHERE account_id=?`, accountID).Scan(&mvStr); err != nil {
		return err
	}
	mv := mustDecimal(mvStr.String, "0")
	var cashStr, frozenStr, initialStr string
	if err := tx.QueryRowContext(ctx, `SELECT available_cash, frozen_cash, initial_cash FROM paper_accounts WHERE id=? FOR UPDATE`, accountID).Scan(&cashStr, &frozenStr, &initialStr); err != nil {
		return err
	}
	total := mustDecimal(cashStr, "0").Add(mustDecimal(frozenStr, "0")).Add(mv)
	profit := total.Sub(mustDecimal(initialStr, "0"))
	if _, err := tx.ExecContext(ctx, `UPDATE paper_accounts SET total_assets=?, total_profit=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, total, profit, accountID); err != nil {
		return err
	}
	_, _ = tx.ExecContext(ctx, `
INSERT INTO paper_equity_curve (account_id, snapshot_time, total_assets, available_cash, position_value, total_profit)
VALUES (?, CURRENT_TIMESTAMP, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE total_assets=VALUES(total_assets), available_cash=VALUES(available_cash), position_value=VALUES(position_value), total_profit=VALUES(total_profit)`, accountID, total, cashStr, mv, profit)
	return nil
}

func (s *TradingService) marketValue(ctx context.Context, accountID uint64) (decimal.Decimal, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(market_value),0) FROM paper_positions WHERE account_id=?`, accountID).Scan(&value)
	return mustDecimal(value, "0"), err
}

func (s *TradingService) latestQuote(ctx context.Context, symbol string) (*quote, error) {
	reply, err := s.market.RealtimeBySymbol(ctx, &marketclient.SymbolRequest{Symbol: symbol})
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data struct {
			Items []struct {
				Price json.RawMessage `json:"price"`
				Name  string          `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(reply.Json), &payload); err != nil {
		return nil, err
	}
	if len(payload.Data.Items) == 0 {
		return nil, fmt.Errorf("empty quote")
	}
	price := decimal.Zero
	if err := json.Unmarshal(payload.Data.Items[0].Price, &price); err != nil {
		var f float64
		if ferr := json.Unmarshal(payload.Data.Items[0].Price, &f); ferr != nil {
			return nil, err
		}
		price = decimal.NewFromFloat(f)
	}
	if !price.GreaterThan(decimal.Zero) {
		return nil, fmt.Errorf("invalid quote price")
	}
	return &quote{Price: price, Name: payload.Data.Items[0].Name}, nil
}

func (s *TradingService) decidePrice(orderType, side, limitPrice string, latest decimal.Decimal) (decimal.Decimal, bool) {
	if orderType == "MARKET" {
		if side == "BUY" {
			return latest.Mul(decimal.NewFromInt(1).Add(s.fee.SlippageRate)), false
		}
		return latest.Mul(decimal.NewFromInt(1).Sub(s.fee.SlippageRate)), false
	}
	limit := mustDecimal(limitPrice, "0")
	if side == "BUY" {
		return limit, latest.GreaterThan(limit)
	}
	return limit, latest.LessThan(limit)
}

func (s *TradingService) feeFor(side string, amount decimal.Decimal) decimal.Decimal {
	rate := s.fee.BuyFeeRate
	if side == "SELL" {
		rate = s.fee.SellFeeRate
	}
	fee := amount.Mul(rate)
	if fee.LessThan(s.fee.MinFee) {
		return s.fee.MinFee
	}
	return fee
}

func (s *TradingService) publish(ctx context.Context, eventType string, userID uint64, payload map[string]any) {
	if s.producer == nil {
		return
	}
	_ = s.producer.Publish(ctx, xmq.Event{EventID: idgen.NewEventID(), EventType: eventType, TraceID: trace.FromContext(ctx), UserID: userID, Payload: payload, CreatedAt: time.Now()})
}

func validateOrder(in *paper.CreateOrderReq) error {
	if in.GetUserId() == 0 || in.GetAccountId() == 0 || strings.TrimSpace(in.GetSymbol()) == "" || strings.TrimSpace(in.GetClientOrderId()) == "" {
		return status.Error(codes.InvalidArgument, "参数错误")
	}
	if in.GetSide() != "BUY" && in.GetSide() != "SELL" {
		return status.Error(codes.InvalidArgument, "side 只能是 BUY 或 SELL")
	}
	if in.GetOrderType() != "MARKET" && in.GetOrderType() != "LIMIT" {
		return status.Error(codes.InvalidArgument, "order_type 只能是 MARKET 或 LIMIT")
	}
	if in.GetVolume() <= 0 {
		return status.Error(codes.InvalidArgument, "数量非法")
	}
	if in.GetSide() == "BUY" && in.GetVolume()%100 != 0 {
		return status.Error(codes.InvalidArgument, "A股买入数量必须为 100 股的整数倍")
	}
	if in.GetOrderType() == "LIMIT" && !mustDecimal(in.GetPrice(), "0").GreaterThan(decimal.Zero) {
		return status.Error(codes.InvalidArgument, "价格非法")
	}
	return nil
}

func scanAccount(row scanner) (*accountRow, error) {
	var a accountRow
	var initial, cash, frozen, total, profit string
	if err := row.Scan(&a.ID, &a.UserID, &a.Name, &initial, &cash, &frozen, &total, &profit, &a.Status, &a.CreatedAt); err != nil {
		return nil, err
	}
	a.InitialCash, a.Cash, a.FrozenCash = mustDecimal(initial, "0"), mustDecimal(cash, "0"), mustDecimal(frozen, "0")
	a.TotalAssets, a.TotalProfit = mustDecimal(total, "0"), mustDecimal(profit, "0")
	return &a, nil
}

func scanOrder(row scanner) (*orderRow, error) {
	var o orderRow
	var price, reserved, avg, amount, fee string
	if err := row.Scan(&o.ID, &o.OrderNo, &o.ClientOrderID, &o.AccountID, &o.UserID, &o.Symbol, &o.Name, &o.Side, &o.OrderType, &price, &o.Quantity, &o.FilledQuantity, &o.RemainingQuantity, &reserved, &avg, &o.Status, &amount, &fee, &o.Remark, &o.ExpireAt, &o.QuoteTime, &o.DataMode, &o.CreatedAt); err != nil {
		return nil, err
	}
	o.Price, o.ReservedCash, o.AvgFillPrice, o.Amount, o.Fee = mustDecimal(price, "0"), mustDecimal(reserved, "0"), mustDecimal(avg, "0"), mustDecimal(amount, "0"), mustDecimal(fee, "0")
	return &o, nil
}

func scanPosition(row scanner) (*paper.Position, error) {
	var id, accountID, userID uint64
	var symbol, name, lastBuyDate, avg, price, mv, profit string
	var qty, avail, frozen, todayBuy int64
	if err := row.Scan(&id, &accountID, &userID, &symbol, &name, &qty, &avail, &frozen, &todayBuy, &lastBuyDate, &avg, &price, &mv, &profit); err != nil {
		return nil, err
	}
	return &paper.Position{Id: id, AccountId: accountID, UserId: userID, Symbol: symbol, Name: name, Volume: qty, AvailableVolume: avail, FrozenVolume: frozen, TodayBuyVolume: todayBuy, LastBuyDate: lastBuyDate, AvgCost: avg, MarketPrice: price, MarketValue: mv, Profit: profit, ProfitRate: "0"}, nil
}

func scanTrade(row scanner) (*paper.Trade, error) {
	var t paper.Trade
	if err := row.Scan(&t.Id, &t.TradeNo, &t.OrderId, &t.OrderNo, &t.AccountId, &t.UserId, &t.Symbol, &t.Name, &t.Side, &t.Price, &t.Volume, &t.Amount, &t.Fee, &t.StampDuty, &t.Slippage, &t.TradeTime, &t.DepthLevel, &t.QuoteTime, &t.DataMode); err != nil {
		return nil, err
	}
	return &t, nil
}

func accountPB(a *accountRow, marketValue decimal.Decimal) *paper.Account {
	rate := decimal.Zero
	if a.InitialCash.GreaterThan(decimal.Zero) {
		rate = a.TotalProfit.Div(a.InitialCash)
	}
	return &paper.Account{Id: a.ID, UserId: a.UserID, AccountName: a.Name, InitialCash: money(a.InitialCash), Cash: money(a.Cash), FrozenCash: money(a.FrozenCash), TotalAsset: money(a.TotalAssets), MarketValue: money(marketValue), TotalProfit: money(a.TotalProfit), TotalProfitRate: rate.StringFixed(6), Status: a.Status, Disclaimer: disclaimer, CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05")}
}

func orderPB(o *orderRow) *paper.Order {
	expireAt := ""
	if o.ExpireAt.Valid {
		expireAt = o.ExpireAt.Time.Format("2006-01-02 15:04:05")
	}
	return &paper.Order{Id: o.ID, OrderNo: o.OrderNo, ClientOrderId: o.ClientOrderID, AccountId: o.AccountID, UserId: o.UserID, Symbol: o.Symbol, Name: o.Name, Side: o.Side, OrderType: o.OrderType, Price: money(o.Price), Volume: o.Quantity, FilledVolume: o.FilledQuantity, AvgFilledPrice: money(o.AvgFillPrice), Status: o.Status, Amount: money(o.Amount), Fee: money(o.Fee), Remark: o.Remark, CreatedAt: o.CreatedAt.Format("2006-01-02 15:04:05"), Disclaimer: disclaimer, RemainingVolume: o.RemainingQuantity, ExpireAt: expireAt, QuoteTime: o.QuoteTime, DataMode: o.DataMode}
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func rollbackOnError(tx *sql.Tx, _ *error) { _ = tx.Rollback() }

func mustDecimal(value, fallback string) decimal.Decimal {
	if value == "" {
		value = fallback
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		d, _ = decimal.NewFromString(fallback)
	}
	return d
}

func money(d decimal.Decimal) string { return d.StringFixed(4) }
func filledVolume(status string, volume int64) int64 {
	if status == "FILLED" {
		return volume
	}
	return 0
}
func avgPrice(status string, price decimal.Decimal) decimal.Decimal {
	if status == "FILLED" {
		return price
	}
	return decimal.Zero
}
