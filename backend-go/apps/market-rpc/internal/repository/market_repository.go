package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type MarketRepository struct {
	db *sql.DB
}

type StockRow struct {
	Symbol      string
	Name        string
	Market      string
	Exchange    string
	Industry    string
	ListingDate sql.NullString
	Source      string
	UpdatedAt   time.Time
}

type QuoteRow struct {
	Symbol        string
	Name          string
	Price         float64
	Open          float64
	High          float64
	Low           float64
	PreClose      float64
	Volume        int64
	Amount        float64
	ChangeAmount  float64
	ChangePercent float64
	TurnoverRate  float64
	Amplitude     float64
	VolumeRatio   float64
	PERatio       float64
	LimitUp       float64
	LimitDown     float64
	QuoteTime     time.Time
	Source        string
	DataMode      string
	UpdatedAt     time.Time
}

type DailyBarRow struct {
	Symbol        string
	TradeDate     string
	Open          float64
	High          float64
	Low           float64
	Close         float64
	PreClose      float64
	Volume        int64
	Amount        float64
	TurnoverRate  float64
	ChangePercent float64
	ChangeAmount  float64
	Amplitude     float64
	Adjust        string
	Source        string
	DataMode      string
	IsRealtime    bool
	BarStatus     string
	QuoteTime     time.Time
}

type SupportResistanceRow struct {
	Symbol             string
	Period             string
	QuoteTime          time.Time
	SupportPrice       float64
	PivotPrice         float64
	ResistancePrice    float64
	SupportStrength    float64
	ResistanceStrength float64
	MethodJSON         string
	AlgorithmVersion   string
	DataSource         string
	ValidUntil         time.Time
}

func NewMarketRepository(db *sql.DB) *MarketRepository {
	return &MarketRepository{db: db}
}

func (r *MarketRepository) SaveSupportResistance(ctx context.Context, item SupportResistanceRow) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO support_resistance_levels(symbol,period,quote_time,support_price,pivot_price,resistance_price,support_strength,resistance_strength,method_json,algorithm_version,data_source,valid_until)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
ON DUPLICATE KEY UPDATE support_price=VALUES(support_price),pivot_price=VALUES(pivot_price),resistance_price=VALUES(resistance_price),support_strength=VALUES(support_strength),resistance_strength=VALUES(resistance_strength),method_json=VALUES(method_json),data_source=VALUES(data_source),valid_until=VALUES(valid_until)`,
		item.Symbol, item.Period, item.QuoteTime, item.SupportPrice, item.PivotPrice, item.ResistancePrice, item.SupportStrength, item.ResistanceStrength, item.MethodJSON, item.AlgorithmVersion, item.DataSource, item.ValidUntil)
	return err
}

func (r *MarketRepository) EnsureSchema(ctx context.Context) {
	_ = r.addColumn(ctx, "stock_daily_bars", "adjust", "VARCHAR(16) NOT NULL DEFAULT 'qfq'")
	_ = r.addColumn(ctx, "stock_daily_bars", "change_amount", "DECIMAL(18,4) DEFAULT NULL")
	_ = r.addColumn(ctx, "stock_daily_bars", "data_mode", "VARCHAR(32) NOT NULL DEFAULT 'real'")
	_ = r.addIndex(ctx, "stock_daily_bars", "uk_symbol_date_adjust", "UNIQUE KEY uk_symbol_date_adjust (symbol, trade_date, adjust)")
	_ = r.addColumn(ctx, "realtime_quotes", "latest_price", "DECIMAL(18,4) DEFAULT NULL")
	_ = r.addColumn(ctx, "realtime_quotes", "change_amount", "DECIMAL(18,4) DEFAULT NULL")
	_ = r.addColumn(ctx, "realtime_quotes", "change_percent", "DECIMAL(10,4) DEFAULT NULL")
	_ = r.addColumn(ctx, "realtime_quotes", "amplitude", "DECIMAL(10,4) DEFAULT NULL")
	_ = r.addColumn(ctx, "realtime_quotes", "volume_ratio", "DECIMAL(10,4) DEFAULT NULL")
	_ = r.addColumn(ctx, "realtime_quotes", "pe_ratio", "DECIMAL(18,4) DEFAULT NULL")
	_ = r.addColumn(ctx, "realtime_quotes", "limit_up", "DECIMAL(18,4) DEFAULT NULL")
	_ = r.addColumn(ctx, "realtime_quotes", "limit_down", "DECIMAL(18,4) DEFAULT NULL")
}

func (r *MarketRepository) ListStocks(ctx context.Context, keyword string, page, pageSize int) ([]StockRow, int, error) {
	page, pageSize = normalizePage(page, pageSize)
	args := []any{}
	where := "deleted_at IS NULL"
	if keyword != "" {
		where += " AND (symbol LIKE ? OR name LIKE ?)"
		like := "%" + keyword + "%"
		args = append(args, like, like)
	}
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM stocks WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, (page-1)*pageSize, pageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT symbol,name,market,COALESCE(exchange,''),COALESCE(industry,''),CAST(listing_date AS CHAR),COALESCE(source,''),updated_at
FROM stocks WHERE `+where+` ORDER BY symbol LIMIT ?,?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []StockRow{}
	for rows.Next() {
		var item StockRow
		if err := rows.Scan(&item.Symbol, &item.Name, &item.Market, &item.Exchange, &item.Industry, &item.ListingDate, &item.Source, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *MarketRepository) UpsertStocks(ctx context.Context, stocks []StockRow) error {
	for _, item := range stocks {
		if strings.TrimSpace(item.Symbol) == "" || strings.TrimSpace(item.Name) == "" {
			continue
		}
		market := valueOr(item.Market, "A股")
		_, err := r.db.ExecContext(ctx, `
INSERT INTO stocks(symbol,name,market,exchange,industry,listing_date,source)
VALUES(?,?,?,?,?,NULLIF(?,''),?)
ON DUPLICATE KEY UPDATE name=VALUES(name),exchange=VALUES(exchange),industry=VALUES(industry),listing_date=VALUES(listing_date),source=VALUES(source),updated_at=CURRENT_TIMESTAMP`,
			item.Symbol, item.Name, market, item.Exchange, item.Industry, nullableString(item.ListingDate), item.Source)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *MarketRepository) GetStock(ctx context.Context, symbol string) (*StockRow, error) {
	rows, _, err := r.ListStocks(ctx, symbol, 1, 20)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Symbol == symbol {
			return &row, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (r *MarketRepository) GetQuote(ctx context.Context, symbol string) (*QuoteRow, error) {
	var item QuoteRow
	err := r.db.QueryRowContext(ctx, `
SELECT symbol,COALESCE(name,''),COALESCE(latest_price,price),COALESCE(open,0),COALESCE(high,0),COALESCE(low,0),COALESCE(pre_close,0),
COALESCE(volume,0),COALESCE(amount,0),COALESCE(change_amount,0),COALESCE(change_percent,change_pct,0),COALESCE(turnover_rate,0),COALESCE(amplitude,0),
COALESCE(volume_ratio,0),COALESCE(pe_ratio,0),COALESCE(limit_up,0),COALESCE(limit_down,0),
quote_time,COALESCE(source,''),COALESCE(data_mode,'real'),updated_at
FROM realtime_quotes WHERE symbol=?`, symbol).Scan(
		&item.Symbol, &item.Name, &item.Price, &item.Open, &item.High, &item.Low, &item.PreClose,
		&item.Volume, &item.Amount, &item.ChangeAmount, &item.ChangePercent, &item.TurnoverRate, &item.Amplitude,
		&item.VolumeRatio, &item.PERatio, &item.LimitUp, &item.LimitDown,
		&item.QuoteTime, &item.Source, &item.DataMode, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *MarketRepository) ListQuotes(ctx context.Context, symbols []string, limit int) ([]QuoteRow, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	args := []any{}
	where := "1=1"
	if len(symbols) > 0 {
		holders := make([]string, 0, len(symbols))
		for _, symbol := range symbols {
			if strings.TrimSpace(symbol) == "" {
				continue
			}
			holders = append(holders, "?")
			args = append(args, symbol)
		}
		if len(holders) > 0 {
			where = "symbol IN (" + strings.Join(holders, ",") + ")"
		}
	}
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, `
SELECT symbol,COALESCE(name,''),COALESCE(latest_price,price),COALESCE(open,0),COALESCE(high,0),COALESCE(low,0),COALESCE(pre_close,0),
COALESCE(volume,0),COALESCE(amount,0),COALESCE(change_amount,0),COALESCE(change_percent,change_pct,0),COALESCE(turnover_rate,0),COALESCE(amplitude,0),
COALESCE(volume_ratio,0),COALESCE(pe_ratio,0),COALESCE(limit_up,0),COALESCE(limit_down,0),
quote_time,COALESCE(source,''),COALESCE(data_mode,'real'),updated_at
FROM realtime_quotes WHERE `+where+` ORDER BY updated_at DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []QuoteRow{}
	for rows.Next() {
		var item QuoteRow
		if err := rows.Scan(&item.Symbol, &item.Name, &item.Price, &item.Open, &item.High, &item.Low, &item.PreClose,
			&item.Volume, &item.Amount, &item.ChangeAmount, &item.ChangePercent, &item.TurnoverRate, &item.Amplitude,
			&item.VolumeRatio, &item.PERatio, &item.LimitUp, &item.LimitDown,
			&item.QuoteTime, &item.Source, &item.DataMode, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *MarketRepository) UpsertQuotes(ctx context.Context, quotes []QuoteRow) error {
	for _, item := range quotes {
		if item.Symbol == "" {
			continue
		}
		if item.QuoteTime.IsZero() {
			item.QuoteTime = time.Now()
		}
		_, err := r.db.ExecContext(ctx, `
INSERT INTO realtime_quotes(symbol,name,price,latest_price,open,high,low,pre_close,volume,amount,change_pct,change_percent,change_amount,turnover_rate,amplitude,volume_ratio,pe_ratio,limit_up,limit_down,quote_time,source,data_mode)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON DUPLICATE KEY UPDATE name=VALUES(name),price=VALUES(price),latest_price=VALUES(latest_price),open=VALUES(open),high=VALUES(high),low=VALUES(low),pre_close=VALUES(pre_close),volume=VALUES(volume),amount=VALUES(amount),change_pct=VALUES(change_pct),change_percent=VALUES(change_percent),change_amount=VALUES(change_amount),turnover_rate=VALUES(turnover_rate),amplitude=VALUES(amplitude),volume_ratio=VALUES(volume_ratio),pe_ratio=VALUES(pe_ratio),limit_up=VALUES(limit_up),limit_down=VALUES(limit_down),quote_time=VALUES(quote_time),source=VALUES(source),data_mode=VALUES(data_mode),updated_at=CURRENT_TIMESTAMP`,
			item.Symbol, item.Name, item.Price, item.Price, item.Open, item.High, item.Low, item.PreClose, item.Volume, item.Amount, item.ChangePercent, item.ChangePercent, item.ChangeAmount, item.TurnoverRate, item.Amplitude, item.VolumeRatio, item.PERatio, item.LimitUp, item.LimitDown, item.QuoteTime, item.Source, valueOr(item.DataMode, "real"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *MarketRepository) ListBars(ctx context.Context, symbol, startDate, endDate, adjust string) ([]DailyBarRow, error) {
	adjust = valueOr(adjust, "qfq")
	args := []any{symbol, adjust}
	where := "symbol=? AND adjust=?"
	if startDate != "" {
		where += " AND trade_date>=?"
		args = append(args, startDate)
	}
	if endDate != "" {
		where += " AND trade_date<=?"
		args = append(args, endDate)
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT symbol,DATE_FORMAT(trade_date,'%Y-%m-%d'),open,high,low,close,COALESCE(pre_close,0),COALESCE(volume,0),COALESCE(amount,0),
COALESCE(turnover_rate,0),COALESCE(change_pct,0),COALESCE(change_amount,0),COALESCE(amplitude,0),COALESCE(adjust,'qfq'),COALESCE(source,''),COALESCE(data_mode,'real')
FROM stock_daily_bars WHERE `+where+` ORDER BY trade_date`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DailyBarRow{}
	for rows.Next() {
		var item DailyBarRow
		if err := rows.Scan(&item.Symbol, &item.TradeDate, &item.Open, &item.High, &item.Low, &item.Close, &item.PreClose, &item.Volume, &item.Amount, &item.TurnoverRate, &item.ChangePercent, &item.ChangeAmount, &item.Amplitude, &item.Adjust, &item.Source, &item.DataMode); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *MarketRepository) UpsertBars(ctx context.Context, bars []DailyBarRow) error {
	for _, item := range bars {
		if item.Symbol == "" || item.TradeDate == "" {
			continue
		}
		_, err := r.db.ExecContext(ctx, `
INSERT INTO stock_daily_bars(symbol,trade_date,open,high,low,close,pre_close,volume,amount,turnover_rate,change_pct,change_amount,amplitude,adjust,source,data_mode)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON DUPLICATE KEY UPDATE open=VALUES(open),high=VALUES(high),low=VALUES(low),close=VALUES(close),pre_close=VALUES(pre_close),volume=VALUES(volume),amount=VALUES(amount),turnover_rate=VALUES(turnover_rate),change_pct=VALUES(change_pct),change_amount=VALUES(change_amount),amplitude=VALUES(amplitude),source=VALUES(source),data_mode=VALUES(data_mode),updated_at=CURRENT_TIMESTAMP`,
			item.Symbol, item.TradeDate, item.Open, item.High, item.Low, item.Close, item.PreClose, item.Volume, item.Amount, item.TurnoverRate, item.ChangePercent, item.ChangeAmount, item.Amplitude, valueOr(item.Adjust, "qfq"), item.Source, valueOr(item.DataMode, "real"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *MarketRepository) addColumn(ctx context.Context, table, column, spec string) error {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?`, table, column).Scan(&exists)
	if err != nil || exists > 0 {
		return err
	}
	_, err = r.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, spec))
	return err
}

func (r *MarketRepository) addIndex(ctx context.Context, table, indexName, spec string) error {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=?`, table, indexName).Scan(&exists)
	if err != nil || exists > 0 {
		return err
	}
	_, err = r.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD %s", table, spec))
	return err
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}
	return page, pageSize
}

func nullableString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
