package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/paper-rpc/paper"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type depthLevel struct {
	Level  int32           `json:"level"`
	Price  decimal.Decimal `json:"price"`
	Volume int64           `json:"volume"`
}

type orderBook struct {
	Symbol    string          `json:"symbol"`
	Name      string          `json:"name"`
	Price     decimal.Decimal `json:"price"`
	QuoteTime string          `json:"quote_time"`
	DataMode  string          `json:"data_mode"`
	Bids      []depthLevel    `json:"bids"`
	Asks      []depthLevel    `json:"asks"`
}

type depthFill struct {
	Level  int32
	Price  decimal.Decimal
	Volume int64
}

func (s *TradingService) loadOrderBook(ctx context.Context, symbol string) (*orderBook, error) {
	reply, err := s.market.OrderBook(ctx, &marketclient.SymbolRequest{Symbol: symbol})
	if err != nil {
		return nil, err
	}
	return parseOrderBook(reply.Json, symbol)
}

func parseOrderBook(raw, symbol string) (*orderBook, error) {
	var envelope struct {
		Code int       `json:"code"`
		Data orderBook `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return nil, err
	}
	book := &envelope.Data
	if envelope.Code != 0 || strings.ToLower(book.DataMode) != "real" || book.QuoteTime == "" {
		return nil, fmt.Errorf("real order book unavailable")
	}
	if strings.TrimSpace(book.Symbol) != strings.TrimSpace(symbol) {
		return nil, fmt.Errorf("order book symbol mismatch: requested=%s received=%s", symbol, book.Symbol)
	}
	if len(book.Bids) == 0 || len(book.Asks) == 0 {
		return nil, fmt.Errorf("empty order book")
	}
	return book, nil
}

func matchDepth(book *orderBook, side, orderType string, limit decimal.Decimal, requested int64) []depthFill {
	levels := book.Asks
	if side == "SELL" {
		levels = book.Bids
	}
	remaining := requested
	result := make([]depthFill, 0, len(levels))
	for _, level := range levels {
		if remaining <= 0 || level.Volume <= 0 || !level.Price.GreaterThan(decimal.Zero) {
			break
		}
		if orderType == "LIMIT" {
			if side == "BUY" && level.Price.GreaterThan(limit) {
				break
			}
			if side == "SELL" && level.Price.LessThan(limit) {
				break
			}
		}
		qty := level.Volume
		if qty > remaining {
			qty = remaining
		}
		result = append(result, depthFill{Level: level.Level, Price: level.Price, Volume: qty})
		remaining -= qty
	}
	return result
}

func fillSummary(fills []depthFill) (int64, decimal.Decimal, decimal.Decimal) {
	var qty int64
	amount := decimal.Zero
	for _, fill := range fills {
		qty += fill.Volume
		amount = amount.Add(fill.Price.Mul(decimal.NewFromInt(fill.Volume)))
	}
	if qty == 0 {
		return 0, decimal.Zero, decimal.Zero
	}
	return qty, amount, amount.Div(decimal.NewFromInt(qty))
}

func consumeDepth(book *orderBook, side string, fills []depthFill) {
	levels := book.Asks
	if side == "SELL" {
		levels = book.Bids
	}
	for _, fill := range fills {
		for index := range levels {
			if levels[index].Level == fill.Level && levels[index].Price.Equal(fill.Price) {
				levels[index].Volume -= fill.Volume
				if levels[index].Volume < 0 {
					levels[index].Volume = 0
				}
				break
			}
		}
	}
	if side == "SELL" {
		book.Bids = levels
	} else {
		book.Asks = levels
	}
}

func (s *TradingService) fillDepthOrder(ctx context.Context, tx *sql.Tx, acc *accountRow, orderNo, name string, book *orderBook, fills []depthFill, in *paper.CreateOrderReq) (*paper.Order, error) {
	filledQty, amount, avgPrice := fillSummary(fills)
	if filledQty <= 0 {
		return nil, status.Error(codes.FailedPrecondition, "当前盘口没有满足价格条件的可见委托")
	}
	remaining := in.GetVolume() - filledQty
	fee := s.feeFor(in.GetSide(), amount)
	stampDuty := decimal.Zero
	if in.GetSide() == "SELL" {
		stampDuty = amount.Mul(s.fee.StampDutyRate)
	}
	reservedCash := decimal.Zero
	statusText := "FILLED"
	remark := "按真实五档盘口逐档成交"
	if remaining > 0 {
		if in.GetOrderType() == "LIMIT" {
			statusText = "PARTIAL_FILLED"
			remark = "盘口部分成交，剩余限价委托等待后续盘口"
			if in.GetSide() == "BUY" {
				limit := mustDecimal(in.GetPrice(), "0")
				reservedCash = limit.Mul(decimal.NewFromInt(remaining)).Add(s.feeFor("BUY", limit.Mul(decimal.NewFromInt(remaining))))
			}
		} else {
			statusText = "PARTIAL_FILLED_CANCELED"
			remark = "五档可见数量不足，市价单剩余数量已取消"
		}
	}

	if in.GetSide() == "BUY" {
		total := amount.Add(fee).Add(reservedCash)
		if acc.Cash.LessThan(total) {
			return nil, status.Error(codes.FailedPrecondition, "资金不足")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash=available_cash-?, frozen_cash=frozen_cash+? WHERE id=?`, total, reservedCash, acc.ID); err != nil {
			return nil, err
		}
		if err := s.addBuyPosition(ctx, tx, in.GetUserId(), in.GetAccountId(), in.GetSymbol(), name, filledQty, avgPrice, amount); err != nil {
			return nil, err
		}
	} else {
		if err := s.reduceSellPosition(ctx, tx, in.GetUserId(), in.GetAccountId(), in.GetSymbol(), filledQty, avgPrice); err != nil {
			return nil, err
		}
		if remaining > 0 && in.GetOrderType() == "LIMIT" {
			if err := s.freezeSellPosition(ctx, tx, in.GetUserId(), in.GetAccountId(), in.GetSymbol(), remaining); err != nil {
				return nil, err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash=available_cash+? WHERE id=?`, amount.Sub(fee).Sub(stampDuty), acc.ID); err != nil {
			return nil, err
		}
	}

	orderPrice := avgPrice
	if in.GetOrderType() == "LIMIT" {
		orderPrice = mustDecimal(in.GetPrice(), "0")
	}
	traceID := trace.FromContext(ctx)
	res, err := tx.ExecContext(ctx, `
INSERT INTO paper_orders (order_no, client_order_id, account_id, user_id, symbol, name, side, order_type, price, quantity, filled_quantity, remaining_quantity, reserved_cash, avg_fill_price, status, amount, fee, slippage, submitted_at, filled_at, expire_at, quote_time, data_mode, trace_id, remark)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP, CASE WHEN ?='FILLED' THEN CURRENT_TIMESTAMP ELSE NULL END, CASE WHEN ?='PARTIAL_FILLED' THEN TIMESTAMP(CURRENT_DATE, '15:00:00') ELSE NULL END, ?, 'real', ?, ?)`,
		orderNo, in.GetClientOrderId(), in.GetAccountId(), in.GetUserId(), in.GetSymbol(), name, in.GetSide(), in.GetOrderType(), orderPrice, in.GetVolume(), filledQty, remaining, reservedCash, avgPrice, statusText, amount, fee, statusText, statusText, book.QuoteTime, traceID, remark)
	if err != nil {
		return nil, err
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	for index, fill := range fills {
		fillAmount := fill.Price.Mul(decimal.NewFromInt(fill.Volume))
		fillFee := fee.Mul(fillAmount).Div(amount)
		fillStamp := stampDuty.Mul(fillAmount).Div(amount)
		basePrice := book.Price
		slippage := fill.Price.Sub(basePrice).Abs().Mul(decimal.NewFromInt(fill.Volume))
		if _, err := tx.ExecContext(ctx, `
INSERT INTO paper_trades (trade_no, account_id, order_id, order_no, user_id, symbol, name, side, price, quantity, depth_level, quote_time, data_mode, trace_id, amount, fee, stamp_duty, slippage, trade_time)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'real', ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			"PT"+idgen.NewEventID()+fmt.Sprintf("%02d", index), in.GetAccountId(), orderID, orderNo, in.GetUserId(), in.GetSymbol(), name, in.GetSide(), fill.Price, fill.Volume, fill.Level, book.QuoteTime, traceID, fillAmount, fillFee, fillStamp, slippage); err != nil {
			return nil, err
		}
	}
	if err := s.refreshAccountTx(ctx, tx, in.GetAccountId()); err != nil {
		return nil, err
	}
	return s.orderPBByID(ctx, tx, in.GetUserId(), uint64(orderID))
}
