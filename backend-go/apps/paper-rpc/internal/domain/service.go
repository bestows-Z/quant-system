package domain

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/paper-rpc/internal/config"
	"quant-system/backend-go/apps/paper-rpc/paper"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/markettime"
	"quant-system/backend-go/common/xmq"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const disclaimer = "paper trading / 模拟交易，仅供研究，不构成投资建议"

type TradingService struct {
	db          *sql.DB
	market      marketclient.Market
	producer    xmq.Producer
	fee         FeeConfig
	quoteMaxAge time.Duration
}

type FeeConfig struct {
	BuyFeeRate    decimal.Decimal
	SellFeeRate   decimal.Decimal
	MinFee        decimal.Decimal
	StampDutyRate decimal.Decimal
	SlippageRate  decimal.Decimal
}

type quote struct {
	Price decimal.Decimal
	Name  string
}

type accountRow struct {
	ID          uint64
	UserID      uint64
	Name        string
	InitialCash decimal.Decimal
	Cash        decimal.Decimal
	FrozenCash  decimal.Decimal
	TotalAssets decimal.Decimal
	TotalProfit decimal.Decimal
	Status      string
	CreatedAt   time.Time
}

type orderRow struct {
	ID                uint64
	OrderNo           string
	ClientOrderID     string
	AccountID         uint64
	UserID            uint64
	Symbol            string
	Name              string
	Side              string
	OrderType         string
	Price             decimal.Decimal
	Quantity          int64
	FilledQuantity    int64
	RemainingQuantity int64
	ReservedCash      decimal.Decimal
	AvgFillPrice      decimal.Decimal
	Status            string
	Amount            decimal.Decimal
	Fee               decimal.Decimal
	Remark            string
	ExpireAt          sql.NullTime
	QuoteTime         string
	DataMode          string
	CreatedAt         time.Time
}

func NewTradingService(db *sql.DB, market marketclient.Market, producer xmq.Producer, c config.Config) *TradingService {
	return &TradingService{
		db:       db,
		market:   market,
		producer: producer,
		fee: FeeConfig{
			BuyFeeRate:    mustDecimal(c.FeeConfig.BuyFeeRate, "0.0003"),
			SellFeeRate:   mustDecimal(c.FeeConfig.SellFeeRate, "0.0003"),
			MinFee:        mustDecimal(c.FeeConfig.MinFee, "5"),
			StampDutyRate: mustDecimal(c.FeeConfig.StampDutyRate, "0.0005"),
			SlippageRate:  mustDecimal(c.SlippageConfig.MarketOrderSlippageRate, "0.0005"),
		},
		quoteMaxAge: time.Duration(c.MarketSafety.MaxQuoteAgeSeconds) * time.Second,
	}
}

func (s *TradingService) CreateAccount(ctx context.Context, in *paper.CreateAccountReq) (*paper.Account, error) {
	if in.GetUserId() == 0 || strings.TrimSpace(in.GetAccountName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "账户名称不能为空")
	}
	initial, err := decimal.NewFromString(in.GetInitialCash())
	if err != nil || !initial.GreaterThan(decimal.Zero) {
		return nil, status.Error(codes.InvalidArgument, "initial_cash 必须大于 0")
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO paper_accounts (user_id, name, initial_cash, available_cash, frozen_cash, total_assets, total_profit, status)
VALUES (?, ?, ?, ?, 0, ?, 0, 'ACTIVE')`, in.GetUserId(), strings.TrimSpace(in.GetAccountName()), initial, initial, initial)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetAccount(ctx, in.GetUserId(), uint64(id))
}

func (s *TradingService) ListAccounts(ctx context.Context, userID uint64) ([]*paper.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, name, initial_cash, available_cash, frozen_cash, total_assets, total_profit, status, created_at
FROM paper_accounts WHERE user_id = ? AND deleted_at IS NULL ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*paper.Account
	for rows.Next() {
		acc, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, accountPB(acc, decimal.Zero))
	}
	return out, rows.Err()
}

func (s *TradingService) GetAccount(ctx context.Context, userID, accountID uint64) (*paper.Account, error) {
	acc, err := s.selectAccount(ctx, s.db, userID, accountID, false)
	if err != nil {
		return nil, err
	}
	mv, _ := s.marketValue(ctx, accountID)
	return accountPB(acc, mv), nil
}

func (s *TradingService) ListPositions(ctx context.Context, userID, accountID uint64) ([]*paper.Position, error) {
	if _, err := s.selectAccount(ctx, s.db, userID, accountID, false); err != nil {
		return nil, err
	}
	if err := s.settlePositions(ctx, userID, accountID, time.Now()); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, account_id, user_id, symbol, COALESCE(name,''), quantity, available_quantity, frozen_quantity,
today_buy_quantity, COALESCE(DATE_FORMAT(last_buy_date, '%Y-%m-%d'), ''), avg_cost, market_price, market_value, unrealized_pnl
FROM paper_positions WHERE user_id = ? AND account_id = ? ORDER BY id DESC`, userID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*paper.Position
	for rows.Next() {
		item, err := scanPosition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *TradingService) CreateOrder(ctx context.Context, in *paper.CreateOrderReq) (*paper.Order, error) {
	if err := validateOrder(in); err != nil {
		return nil, err
	}
	if existing, err := s.orderByClientID(ctx, in.GetUserId(), in.GetClientOrderId()); err == nil && existing != nil {
		return existing, nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	book, bookErr := s.loadOrderBook(ctx, in.GetSymbol())
	if bookErr != nil {
		return nil, status.Error(codes.Unavailable, "真实五档盘口不可用，模拟委托已拒绝")
	}
	if err := markettime.ValidateFreshQuote(time.Now(), book.QuoteTime, s.quoteMaxAge); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	q := &quote{Price: book.Price, Name: book.Name}
	name := strings.TrimSpace(in.GetName())
	if name == "" {
		name = q.Name
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackOnError(tx, &err)

	acc, err := s.selectAccount(ctx, tx, in.GetUserId(), in.GetAccountId(), true)
	if err != nil {
		return nil, err
	}
	if acc.Status != "ACTIVE" {
		return nil, status.Error(codes.FailedPrecondition, "账户状态异常")
	}
	if err := s.settlePositionsTx(ctx, tx, in.GetUserId(), in.GetAccountId(), time.Now()); err != nil {
		return nil, err
	}

	orderNo := "PO" + idgen.NewEventID()
	side, orderType := in.GetSide(), in.GetOrderType()
	if book != nil {
		limit := mustDecimal(in.GetPrice(), "0")
		fills := matchDepth(book, side, orderType, limit, in.GetVolume())
		if len(fills) > 0 {
			order, fillErr := s.fillDepthOrder(ctx, tx, acc, orderNo, name, book, fills, in)
			if fillErr != nil {
				return nil, fillErr
			}
			if err = tx.Commit(); err != nil {
				return nil, err
			}
			s.publish(ctx, constants.EventPaperOrderCreated, in.GetUserId(), map[string]any{"order_no": order.OrderNo, "status": order.Status})
			if order.FilledVolume > 0 {
				s.publish(ctx, constants.EventPaperOrderFilled, in.GetUserId(), map[string]any{"order_no": order.OrderNo, "symbol": order.Symbol, "filled_volume": order.FilledVolume})
			}
			return order, nil
		}
		if orderType == "MARKET" {
			return nil, status.Error(codes.FailedPrecondition, "当前五档盘口没有可成交的真实委托，市价单已拒绝")
		}
	}
	fillPrice, pending := s.decidePrice(orderType, side, in.GetPrice(), q.Price)
	amount := fillPrice.Mul(decimal.NewFromInt(in.GetVolume()))
	fee := s.feeFor(side, amount)
	stampDuty := decimal.Zero
	if side == "SELL" {
		stampDuty = amount.Mul(s.fee.StampDutyRate)
	}
	slippage := q.Price.Sub(fillPrice).Abs().Mul(decimal.NewFromInt(in.GetVolume()))

	if pending {
		order, err := s.createPendingOrder(ctx, tx, acc, orderNo, name, fillPrice, amount, fee, in)
		if err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		s.publish(ctx, constants.EventPaperOrderCreated, in.GetUserId(), map[string]any{"order_no": order.OrderNo, "status": order.Status})
		return order, nil
	}

	order, err := s.fillOrder(ctx, tx, acc, orderNo, name, fillPrice, amount, fee, stampDuty, slippage, in)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	s.publish(ctx, constants.EventPaperOrderCreated, in.GetUserId(), map[string]any{"order_no": order.OrderNo, "status": order.Status})
	s.publish(ctx, constants.EventPaperOrderFilled, in.GetUserId(), map[string]any{"order_no": order.OrderNo, "symbol": order.Symbol})
	return order, nil
}

func (s *TradingService) CancelOrder(ctx context.Context, userID, orderID uint64) (*paper.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackOnError(tx, &err)
	o, err := s.selectOrder(ctx, tx, userID, orderID, true)
	if err != nil {
		return nil, err
	}
	if o.Status != "PENDING" && o.Status != "PARTIAL_FILLED" {
		return nil, status.Error(codes.FailedPrecondition, "订单状态不允许撤销")
	}
	if _, err = s.selectAccount(ctx, tx, userID, o.AccountID, true); err != nil {
		return nil, err
	}
	remaining := decimal.NewFromInt(o.Quantity - o.FilledQuantity)
	if o.Side == "BUY" {
		release := o.ReservedCash
		_, err = tx.ExecContext(ctx, `UPDATE paper_accounts SET available_cash = available_cash + ?, frozen_cash = frozen_cash - ? WHERE id = ?`, release, release, o.AccountID)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE paper_positions SET available_quantity = available_quantity + ?, frozen_quantity = frozen_quantity - ? WHERE account_id = ? AND symbol = ?`, remaining.IntPart(), remaining.IntPart(), o.AccountID, o.Symbol)
	}
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE paper_orders SET status='CANCELED', canceled_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, o.ID)
	if err != nil {
		return nil, err
	}
	if err = s.refreshAccountTx(ctx, tx, o.AccountID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	out, err := s.GetOrder(ctx, userID, orderID)
	if err == nil {
		s.publish(ctx, constants.EventPaperOrderCanceled, userID, map[string]any{"order_no": out.OrderNo})
	}
	return out, err
}

func (s *TradingService) ListOrders(ctx context.Context, userID, accountID uint64, symbol, statusText string) ([]*paper.Order, error) {
	query := `SELECT id, order_no, client_order_id, account_id, user_id, symbol, COALESCE(name,''), side, order_type, COALESCE(price,0), quantity, filled_quantity, remaining_quantity, reserved_cash, COALESCE(avg_fill_price,0), status, amount, fee, COALESCE(remark,''), expire_at, quote_time, data_mode, created_at FROM paper_orders WHERE user_id = ?`
	args := []any{userID}
	if accountID > 0 {
		query += ` AND account_id = ?`
		args = append(args, accountID)
	}
	if symbol != "" {
		query += ` AND symbol = ?`
		args = append(args, symbol)
	}
	if statusText != "" {
		query += ` AND status = ?`
		args = append(args, statusText)
	}
	query += ` ORDER BY id DESC LIMIT 200`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*paper.Order
	for rows.Next() {
		item, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, orderPB(item))
	}
	return out, rows.Err()
}

func (s *TradingService) GetOrder(ctx context.Context, userID, orderID uint64) (*paper.Order, error) {
	o, err := s.selectOrder(ctx, s.db, userID, orderID, false)
	if err != nil {
		return nil, err
	}
	return orderPB(o), nil
}

func (s *TradingService) ListTrades(ctx context.Context, userID, accountID uint64, symbol string) ([]*paper.Trade, error) {
	query := `SELECT id, trade_no, order_id, order_no, account_id, user_id, symbol, COALESCE(name,''), side, price, quantity, amount, fee, stamp_duty, slippage, trade_time, depth_level, quote_time, data_mode FROM paper_trades WHERE user_id = ?`
	args := []any{userID}
	if accountID > 0 {
		query += ` AND account_id = ?`
		args = append(args, accountID)
	}
	if symbol != "" {
		query += ` AND symbol = ?`
		args = append(args, symbol)
	}
	query += ` ORDER BY id DESC LIMIT 200`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*paper.Trade
	for rows.Next() {
		item, err := scanTrade(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *TradingService) EquityCurve(ctx context.Context, userID, accountID uint64) ([]*paper.EquityPoint, error) {
	if _, err := s.selectAccount(ctx, s.db, userID, accountID, false); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT DATE(snapshot_time), total_assets, available_cash, position_value, total_profit FROM paper_equity_curve WHERE account_id=? ORDER BY snapshot_time`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*paper.EquityPoint
	for rows.Next() {
		var d string
		var total, cash, mv, profit string
		if err := rows.Scan(&d, &total, &cash, &mv, &profit); err != nil {
			return nil, err
		}
		out = append(out, &paper.EquityPoint{TradeDate: d, TotalAsset: total, Cash: cash, MarketValue: mv, TotalProfit: profit, TotalProfitRate: "0", Drawdown: "0"})
	}
	return out, rows.Err()
}
