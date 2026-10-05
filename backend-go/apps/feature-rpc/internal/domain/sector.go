package domain

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"quant-system/backend-go/common/markettime"

	"github.com/zeromicro/go-zero/core/logx"
)

func (s *Service) EnsureSectorSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS sector_minute_snapshots (
id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, sector_code VARCHAR(160) NOT NULL, sector_name VARCHAR(128) NOT NULL,
snapshot_time DATETIME NOT NULL, change_percent DECIMAL(12,8) DEFAULT NULL, rise_count INT NOT NULL DEFAULT 0,
fall_count INT NOT NULL DEFAULT 0, quoted_count INT NOT NULL DEFAULT 0, stock_count INT NOT NULL DEFAULT 0,
coverage_rate DECIMAL(12,8) NOT NULL DEFAULT 0, total_amount DECIMAL(24,4) NOT NULL DEFAULT 0,
source VARCHAR(64) NOT NULL DEFAULT 'realtime_quotes', data_mode VARCHAR(16) NOT NULL DEFAULT 'real', created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
UNIQUE KEY uk_sector_snapshot (sector_code,snapshot_time), KEY idx_sector_time (sector_code,snapshot_time DESC), KEY idx_snapshot_time (snapshot_time DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

func (s *Service) SectorDetail(ctx context.Context, code string) (string, error) {
	name, err := sectorName(code)
	if err != nil {
		return "", err
	}
	rows, err := s.loadSectorUniverse(ctx, name)
	if err != nil {
		return "", err
	}
	items := calculateSectors(rows)
	if len(items) == 0 {
		return "", sql.ErrNoRows
	}
	return envelope(map[string]any{"sector": items[0], "disclaimer": disclaimer})
}

func (s *Service) SectorConstituents(ctx context.Context, code, sortBy, order string, limit int32) (string, error) {
	name, err := sectorName(code)
	if err != nil {
		return "", err
	}
	rows, err := s.loadSectorUniverse(ctx, name)
	if err != nil {
		return "", err
	}
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	sortQuotes(rows, sortBy, order)
	if len(rows) > int(limit) {
		rows = rows[:limit]
	}
	quoted := 0
	for _, row := range rows {
		if hasQuote(row) {
			quoted++
		}
	}
	coverage := 0.0
	if len(rows) > 0 {
		coverage = float64(quoted) / float64(len(rows))
	}
	statusText, dataMode := "INSUFFICIENT_COVERAGE", "unavailable"
	if sectors := calculateSectors(rows); len(sectors) > 0 {
		statusText, dataMode, coverage, quoted = sectors[0].Status, sectors[0].DataMode, sectors[0].CoverageRate, sectors[0].QuotedCount
	}
	return envelope(map[string]any{"sector_code": code, "sector_name": name, "sector_type": "industry", "items": rows, "count": len(rows), "quoted_count": quoted, "coverage_rate": coverage, "status": statusText, "source": "stocks+realtime_quotes", "data_mode": dataMode, "methodology": "equal_weight"})
}

func (s *Service) SectorTimeshare(ctx context.Context, code string) (string, error) {
	name, err := sectorName(code)
	if err != nil {
		return "", err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT snapshot_time,change_percent,rise_count,fall_count,quoted_count,stock_count,coverage_rate,total_amount,source,data_mode FROM sector_minute_snapshots WHERE sector_code=? AND DATE(snapshot_time)=CURRENT_DATE ORDER BY snapshot_time`, code)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var snapshot time.Time
		var change sql.NullFloat64
		var rise, fall, quoted, total int
		var coverage, amount float64
		var source, dataMode string
		if err = rows.Scan(&snapshot, &change, &rise, &fall, &quoted, &total, &coverage, &amount, &source, &dataMode); err != nil {
			return "", err
		}
		item := map[string]any{"time": snapshot.Format("2006-01-02 15:04"), "rise_count": rise, "fall_count": fall, "quoted_count": quoted, "stock_count": total, "coverage_rate": coverage, "total_amount": amount, "source": source, "data_mode": dataMode}
		if change.Valid {
			item["change_percent"] = change.Float64
		} else {
			item["change_percent"] = nil
		}
		items = append(items, item)
	}
	return envelope(map[string]any{"sector_code": code, "sector_name": name, "items": items, "count": len(items), "source": "sector_minute_snapshots", "data_mode": "real"})
}

func (s *Service) loadSectorUniverse(ctx context.Context, industry string) ([]quoteRow, error) {
	query := `SELECT s.symbol,s.name,COALESCE(s.industry,'未分类'),COALESCE(q.source,'stocks'),COALESCE(q.data_mode,'real'),COALESCE(q.latest_price,q.price,0),COALESCE(q.change_percent,q.change_pct,0),COALESCE(q.volume,0),COALESCE(q.amount,0),COALESCE(q.turnover_rate,0),COALESCE(q.amplitude,0),q.quote_time FROM stocks s LEFT JOIN realtime_quotes q ON q.symbol=s.symbol WHERE s.deleted_at IS NULL AND COALESCE(s.industry,'')<>''`
	args := []any{}
	if industry != "" {
		query += ` AND s.industry=?`
		args = append(args, industry)
	}
	query += ` ORDER BY s.symbol`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]quoteRow, 0)
	for rows.Next() {
		var row quoteRow
		var quoteTime sql.NullTime
		if err = rows.Scan(&row.Symbol, &row.Name, &row.Industry, &row.Source, &row.DataMode, &row.Price, &row.Change, &row.Volume, &row.Amount, &row.Turnover, &row.Amplitude, &quoteTime); err != nil {
			return nil, err
		}
		if quoteTime.Valid {
			row.QuoteTime = quoteTime.Time
		}
		row.Change, row.Turnover, row.Amplitude = normalizePct(row.Change), normalizePct(row.Turnover), normalizePct(row.Amplitude)
		items = append(items, row)
	}
	return items, rows.Err()
}

func sectorName(code string) (string, error) {
	raw := strings.TrimPrefix(strings.TrimSpace(code), "SEC-")
	decoded, err := hex.DecodeString(raw)
	if err != nil || len(decoded) == 0 {
		return "", fmt.Errorf("invalid sector code")
	}
	return string(decoded), nil
}

func (s *Service) StartSectorSnapshotScheduler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				if markettime.IsContinuousAuction(now) {
					if err := s.snapshotSectors(ctx, now); err != nil {
						logx.Errorf("sector snapshot failed: %v", err)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Service) snapshotSectors(ctx context.Context, now time.Time) error {
	rows, err := s.loadSectorUniverse(ctx, "")
	if err != nil {
		return err
	}
	items := calculateSectors(rows)
	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	snapshot := now.Truncate(time.Minute)
	for _, item := range items {
		var change any
		if item.QuotedCount > 0 {
			change = item.AverageChange
		}
		if _, err = s.db.ExecContext(ctx, `INSERT INTO sector_minute_snapshots(sector_code,sector_name,snapshot_time,change_percent,rise_count,fall_count,quoted_count,stock_count,coverage_rate,total_amount,source,data_mode) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE change_percent=VALUES(change_percent),rise_count=VALUES(rise_count),fall_count=VALUES(fall_count),quoted_count=VALUES(quoted_count),stock_count=VALUES(stock_count),coverage_rate=VALUES(coverage_rate),total_amount=VALUES(total_amount)`, item.Code, item.Name, snapshot, change, item.RiseCount, item.FallCount, item.QuotedCount, item.StockCount, item.CoverageRate, item.TotalAmount, item.Source, item.DataMode); err != nil {
			return err
		}
	}
	return nil
}
