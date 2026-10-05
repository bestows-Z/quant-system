package domain

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/markettime"
	"quant-system/backend-go/common/trace"

	"github.com/shopspring/decimal"
)

func (s *TradingService) MatchPendingBySymbol(ctx context.Context, symbol, orderBookJSON string) (int64, int64, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return 0, 0, fmt.Errorf("symbol is required")
	}
	var book *orderBook
	var err error
	if strings.TrimSpace(orderBookJSON) != "" {
		book, err = parseOrderBook(orderBookJSON, symbol)
	} else {
		book, err = s.loadOrderBook(ctx, symbol)
	}
	if err != nil {
		return 0, 0, err
	}
	if err := markettime.ValidateFreshQuote(time.Now(), book.QuoteTime, s.quoteMaxAge); err != nil {
		return 0, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id FROM paper_orders
WHERE symbol=? AND status IN ('PENDING','PARTIAL_FILLED') AND remaining_quantity>0
  AND (expire_at IS NULL OR expire_at>CURRENT_TIMESTAMP)
ORDER BY created_at, id LIMIT 200`, symbol)
	if err != nil {
		return 0, 0, err
	}
	type pendingID struct{ id, userID uint64 }
	items := make([]pendingID, 0)
	for rows.Next() {
		var item pendingID
		if scanErr := rows.Scan(&item.id, &item.userID); scanErr != nil {
			rows.Close()
			return 0, 0, scanErr
		}
		items = append(items, item)
	}
	if err = rows.Close(); err != nil {
		return 0, 0, err
	}
	var matched, trades int64
	for _, item := range items {
		fillCount, matchErr := s.continuePendingOrder(ctx, item.userID, item.id, book)
		if matchErr != nil {
			return matched, trades, matchErr
		}
		if fillCount > 0 {
			matched++
			trades += fillCount
		}
	}
	return matched, trades, nil
}

func (s *TradingService) continuePendingOrder(ctx context.Context, userID, orderID uint64, book *orderBook) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollbackOnError(tx, &err)
	o, err := s.selectOrder(ctx, tx, userID, orderID, true)
	if err != nil {
		return 0, err
	}
	if (o.Status != "PENDING" && o.Status != "PARTIAL_FILLED") || o.RemainingQuantity <= 0 || !newerQuoteTime(o.QuoteTime, book.QuoteTime) {
		_ = tx.Rollback()
		return 0, nil
	}
	acc, err := s.selectAccount(ctx, tx, userID, o.AccountID, true)
	if err != nil {
		return 0, err
	}
	fills := matchDepth(book, o.Side, "LIMIT", o.Price, o.RemainingQuantity)
	filledQty, fillAmount, fillAvg := fillSummary(fills)
	if filledQty <= 0 {
		_, err = tx.ExecContext(ctx, `UPDATE paper_orders SET quote_time=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, book.QuoteTime, o.ID)
		if err == nil {
			err = tx.Commit()
		}
		return 0, err
	}
	previousAmount, previousFee := o.Amount, o.Fee
	if o.FilledQuantity == 0 {
		previousAmount, previousFee = decimal.Zero, decimal.Zero
	}
	cumulativeAmount := previousAmount.Add(fillAmount)
	cumulativeFee := s.feeFor(o.Side, cumulativeAmount)
	incrementalFee := cumulativeFee.Sub(previousFee)
	if incrementalFee.IsNegative() {
		incrementalFee = decimal.Zero
	}
	newFilled := o.FilledQuantity + filledQty
	remaining := o.Quantity - newFilled
	avgPrice := cumulativeAmount.Div(decimal.NewFromInt(newFilled))
	statusText := "PARTIAL_FILLED"
	filledAt := "NULL"
	if remaining == 0 {
		statusText = "FILLED"
		filledAt = "CURRENT_TIMESTAMP"
	}

	newReserve := decimal.Zero
	if o.Side == "BUY" {
		if remaining > 0 {
			finalAmount := cumulativeAmount.Add(o.Price.Mul(decimal.NewFromInt(remaining)))
			remainingFee := s.feeFor("BUY", finalAmount).Sub(cumulativeFee)
			if remainingFee.IsNegative() {
				remainingFee = decimal.Zero
			}
			newReserve = o.Price.Mul(decimal.NewFromInt(remaining)).Add(remainingFee)
		}
		release := o.ReservedCash.Sub(fillAmount).Sub(incrementalFee).Sub(newReserve)
		if release.IsNegative() && acc.Cash.LessThan(release.Abs()) {
			return 0, fmt.Errorf("续撮资金不足")
		}
		if _, err = tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash=available_cash+?, frozen_cash=frozen_cash-?+? WHERE id=?`, release, o.ReservedCash, newReserve, o.AccountID); err != nil {
			return 0, err
		}
		if err = s.addBuyPosition(ctx, tx, o.UserID, o.AccountID, o.Symbol, o.Name, filledQty, fillAvg, fillAmount); err != nil {
			return 0, err
		}
	} else {
		stampDuty := fillAmount.Mul(s.fee.StampDutyRate)
		if err = s.reduceFrozenSellPosition(ctx, tx, o, filledQty, fillAvg); err != nil {
			return 0, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash=available_cash+? WHERE id=?`, fillAmount.Sub(incrementalFee).Sub(stampDuty), o.AccountID); err != nil {
			return 0, err
		}
	}

	query := fmt.Sprintf(`UPDATE paper_orders SET filled_quantity=?, remaining_quantity=?, reserved_cash=?, avg_fill_price=?, status=?, amount=?, fee=?, quote_time=?, filled_at=%s, remark=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, filledAt)
	remark := "真实五档盘口续撮"
	if remaining == 0 {
		remark = "真实五档盘口续撮完成"
	}
	if _, err = tx.ExecContext(ctx, query, newFilled, remaining, newReserve, avgPrice, statusText, cumulativeAmount, cumulativeFee, book.QuoteTime, remark, o.ID); err != nil {
		return 0, err
	}
	for index, fill := range fills {
		fillValue := fill.Price.Mul(decimal.NewFromInt(fill.Volume))
		feePart := incrementalFee.Mul(fillValue).Div(fillAmount)
		stampPart := decimal.Zero
		if o.Side == "SELL" {
			stampPart = fillValue.Mul(s.fee.StampDutyRate)
		}
		if _, err = tx.ExecContext(ctx, `
INSERT INTO paper_trades (trade_no, account_id, order_id, order_no, user_id, symbol, name, side, price, quantity, depth_level, quote_time, data_mode, trace_id, amount, fee, stamp_duty, slippage, trade_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'real', ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)`,
			"PT"+idgen.NewEventID()+fmt.Sprintf("%02d", index), o.AccountID, o.ID, o.OrderNo, o.UserID, o.Symbol, o.Name, o.Side, fill.Price, fill.Volume, fill.Level, book.QuoteTime, trace.FromContext(ctx), fillValue, feePart, stampPart); err != nil {
			return 0, err
		}
	}
	if err = s.refreshAccountTx(ctx, tx, o.AccountID); err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	s.publish(ctx, constants.EventPaperOrderFilled, o.UserID, map[string]any{"order_no": o.OrderNo, "symbol": o.Symbol, "filled_volume": filledQty, "status": statusText})
	consumeDepth(book, o.Side, fills)
	return int64(len(fills)), nil
}

func (s *TradingService) reduceFrozenSellPosition(ctx context.Context, tx *sql.Tx, o *orderRow, volume int64, price decimal.Decimal) error {
	var quantity, frozen int64
	if err := tx.QueryRowContext(ctx, `SELECT quantity, frozen_quantity FROM paper_positions WHERE user_id=? AND account_id=? AND symbol=? FOR UPDATE`, o.UserID, o.AccountID, o.Symbol).Scan(&quantity, &frozen); err != nil {
		return err
	}
	if quantity < volume || frozen < volume {
		return fmt.Errorf("冻结持仓不足")
	}
	marketValue := price.Mul(decimal.NewFromInt(quantity - volume))
	_, err := tx.ExecContext(ctx, `UPDATE paper_positions SET quantity=quantity-?, frozen_quantity=frozen_quantity-?, market_price=?, market_value=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND account_id=? AND symbol=?`, volume, volume, price, marketValue, o.UserID, o.AccountID, o.Symbol)
	return err
}

func (s *TradingService) ExpireOrders(ctx context.Context, limit int32) (int64, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id FROM paper_orders WHERE status IN ('PENDING','PARTIAL_FILLED') AND expire_at IS NOT NULL AND expire_at<=CURRENT_TIMESTAMP ORDER BY expire_at, id LIMIT ?`, limit)
	if err != nil {
		return 0, err
	}
	type expiredID struct{ id, userID uint64 }
	items := make([]expiredID, 0)
	for rows.Next() {
		var item expiredID
		if err = rows.Scan(&item.id, &item.userID); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, item)
	}
	rows.Close()
	var count int64
	for _, item := range items {
		if err = s.expireOrder(ctx, item.userID, item.id); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *TradingService) expireOrder(ctx context.Context, userID, orderID uint64) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackOnError(tx, &err)
	o, err := s.selectOrder(ctx, tx, userID, orderID, true)
	if err != nil {
		return err
	}
	if o.Status != "PENDING" && o.Status != "PARTIAL_FILLED" {
		_ = tx.Rollback()
		return nil
	}
	if o.Side == "BUY" {
		_, err = tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash=available_cash+?, frozen_cash=frozen_cash-? WHERE id=?`, o.ReservedCash, o.ReservedCash, o.AccountID)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE paper_positions SET available_quantity=available_quantity+?, frozen_quantity=frozen_quantity-? WHERE account_id=? AND symbol=?`, o.RemainingQuantity, o.RemainingQuantity, o.AccountID, o.Symbol)
	}
	if err != nil {
		return err
	}
	statusText := "EXPIRED"
	if o.FilledQuantity > 0 {
		statusText = "PARTIAL_FILLED_CANCELED"
	}
	if _, err = tx.ExecContext(ctx, `UPDATE paper_orders SET status=?, remaining_quantity=0, reserved_cash=0, remark='限价委托到期，剩余部分已撤销', updated_at=CURRENT_TIMESTAMP WHERE id=?`, statusText, o.ID); err != nil {
		return err
	}
	if err = s.refreshAccountTx(ctx, tx, o.AccountID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	s.publish(ctx, constants.EventPaperOrderCanceled, o.UserID, map[string]any{"order_no": o.OrderNo, "status": statusText})
	return nil
}

func newerQuoteTime(previous, current string) bool {
	currentAt, err := markettime.ParseQuoteTime(current)
	if err != nil {
		return false
	}
	previousAt, err := markettime.ParseQuoteTime(previous)
	if err != nil {
		return strings.TrimSpace(previous) != strings.TrimSpace(current)
	}
	return currentAt.After(previousAt)
}
