package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"quant-system/backend-go/common/markettime"
)

const (
	factorVersion  = "market-factor-v1"
	scoringVersion = "candidate-score-v1"
	disclaimer     = "智能候选池仅用于量化研究，不构成投资建议"
)

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

type quoteRow struct {
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	Industry  string    `json:"industry"`
	Source    string    `json:"source"`
	DataMode  string    `json:"data_mode"`
	Price     float64   `json:"price"`
	Change    float64   `json:"change_percent"`
	Volume    float64   `json:"volume"`
	Amount    float64   `json:"amount"`
	Turnover  float64   `json:"turnover_rate"`
	Amplitude float64   `json:"amplitude"`
	QuoteTime time.Time `json:"quote_time"`
}

type overview struct {
	QuoteTime         string  `json:"quote_time"`
	Total             int     `json:"total"`
	RiseCount         int     `json:"rise_count"`
	FallCount         int     `json:"fall_count"`
	FlatCount         int     `json:"flat_count"`
	LimitUpCount      int     `json:"limit_up_count"`
	LimitDownCount    int     `json:"limit_down_count"`
	RiseRatio         float64 `json:"rise_ratio"`
	MedianChange      float64 `json:"median_change"`
	AverageChange     float64 `json:"average_change"`
	TotalAmount       float64 `json:"total_amount"`
	MarketTemperature float64 `json:"market_temperature"`
	Source            string  `json:"source"`
	DataMode          string  `json:"data_mode"`
	Disclaimer        string  `json:"disclaimer"`
}

type sector struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	StockCount    int     `json:"stock_count"`
	RiseCount     int     `json:"rise_count"`
	FallCount     int     `json:"fall_count"`
	AverageChange float64 `json:"average_change"`
	TotalAmount   float64 `json:"total_amount"`
	LeaderSymbol  string  `json:"leader_symbol"`
	LeaderName    string  `json:"leader_name"`
	LeaderChange  float64 `json:"leader_change"`
	StrengthScore float64 `json:"strength_score"`
	QuotedCount   int     `json:"quoted_count"`
	CoverageRate  float64 `json:"coverage_rate"`
	QuoteTime     string  `json:"quote_time"`
	Status        string  `json:"status"`
	Source        string  `json:"source"`
	DataMode      string  `json:"data_mode"`
	Methodology   string  `json:"methodology"`
}

type candidate struct {
	Symbol          string         `json:"symbol"`
	Name            string         `json:"name"`
	Industry        string         `json:"industry"`
	Price           float64        `json:"price"`
	ChangePercent   float64        `json:"change_percent"`
	Amount          float64        `json:"amount"`
	TurnoverRate    float64        `json:"turnover_rate"`
	TotalScore      float64        `json:"total_score"`
	TechnicalScore  float64        `json:"technical_score"`
	CapitalScore    float64        `json:"capital_score"`
	SentimentScore  float64        `json:"sentiment_score"`
	SectorScore     float64        `json:"sector_score"`
	RiskScore       float64        `json:"risk_score"`
	Confidence      float64        `json:"confidence"`
	Reasons         []string       `json:"reasons"`
	NegativeFactors []string       `json:"negative_factors"`
	Evidence        map[string]any `json:"evidence"`
	FactorVersion   string         `json:"factor_version"`
	ScoringVersion  string         `json:"scoring_version"`
	QuoteTime       string         `json:"quote_time"`
	DataSource      string         `json:"data_source"`
	DataMode        string         `json:"data_mode"`
	Disclaimer      string         `json:"disclaimer"`
}

func (s *Service) MarketOverview(ctx context.Context) (string, error) {
	rows, err := s.loadQuotes(ctx, "", 10000)
	if err != nil {
		return "", err
	}
	o := calculateOverview(rows)
	if len(rows) > 0 {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO market_breadth_snapshots(quote_time,rise_count,fall_count,flat_count,limit_up_count,limit_down_count,total_amount,median_change,source) VALUES(?,?,?,?,?,?,?,?,?)`, rows[0].QuoteTime, o.RiseCount, o.FallCount, o.FlatCount, o.LimitUpCount, o.LimitDownCount, o.TotalAmount, o.MedianChange, o.Source)
	}
	return envelope(map[string]any{"overview": o, "distribution": distribution(rows)})
}

func (s *Service) Sectors(ctx context.Context, sectorType, sortBy, order string) (string, error) {
	sectorType = strings.ToLower(strings.TrimSpace(sectorType))
	if sectorType == "" {
		sectorType = "industry"
	}
	if sectorType == "concept" {
		return envelope(map[string]any{"items": []sector{}, "count": 0, "sector_type": "concept", "source_status": "SOURCE_PENDING", "source": "none", "data_mode": "unavailable", "disclaimer": disclaimer})
	}
	if sectorType != "industry" {
		return "", fmt.Errorf("unsupported sector type: %s", sectorType)
	}
	rows, err := s.loadSectorUniverse(ctx, "")
	if err != nil {
		return "", err
	}
	items := calculateSectors(rows)
	sortSectors(items, sortBy, order)
	dataMode := "unavailable"
	for _, item := range items {
		if item.DataMode == "real" {
			dataMode = "real"
			break
		}
	}
	return envelope(map[string]any{"items": items, "count": len(items), "sector_type": "industry", "source_status": "AVAILABLE", "source": "stocks+realtime_quotes", "data_mode": dataMode, "quote_time": latestQuoteTime(rows), "methodology": "equal_weight", "disclaimer": disclaimer})
}

func sortSectors(items []sector, sortBy, order string) {
	desc := strings.ToLower(order) != "asc"
	less := func(i, j int) bool {
		left, right := 0.0, 0.0
		switch sortBy {
		case "change_percent", "average_change":
			left, right = items[i].AverageChange, items[j].AverageChange
		case "amount", "total_amount":
			left, right = items[i].TotalAmount, items[j].TotalAmount
		case "rise_ratio":
			if items[i].QuotedCount > 0 {
				left = float64(items[i].RiseCount) / float64(items[i].QuotedCount)
			}
			if items[j].QuotedCount > 0 {
				right = float64(items[j].RiseCount) / float64(items[j].QuotedCount)
			}
		default:
			left, right = items[i].StrengthScore, items[j].StrengthScore
		}
		if desc {
			return left > right
		}
		return left < right
	}
	sort.SliceStable(items, less)
}

func (s *Service) Screener(ctx context.Context, sortBy, order, industry string, minChange, maxChange, minAmount float64, limit int32) (string, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.loadQuotes(ctx, industry, int(limit)*4)
	if err != nil {
		return "", err
	}
	filtered := make([]quoteRow, 0, len(rows))
	for _, row := range rows {
		if minChange != 0 && row.Change < minChange {
			continue
		}
		if maxChange != 0 && row.Change > maxChange {
			continue
		}
		if minAmount > 0 && row.Amount < minAmount {
			continue
		}
		filtered = append(filtered, row)
	}
	sortQuotes(filtered, sortBy, order)
	if len(filtered) > int(limit) {
		filtered = filtered[:limit]
	}
	return envelope(map[string]any{"items": filtered, "count": len(filtered), "sort_by": normalizeSort(sortBy), "order": normalizeOrder(order), "source": "database", "data_mode": "real"})
}

func (s *Service) Candidates(ctx context.Context, sortBy, industry string, minConfidence float64, limit int32, symbol string, persist bool) (string, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if !persist {
		cached, found, err := s.latestCandidateSnapshot(ctx, sortBy, industry, minConfidence, limit, symbol)
		if err != nil {
			return "", err
		}
		if found {
			return cached, nil
		}
	}
	rows, err := s.loadQuotes(ctx, industry, 300)
	if err != nil {
		return "", err
	}
	sectorScores := sectorScoreMap(calculateSectors(rows))
	items := make([]candidate, 0, len(rows))
	for _, row := range rows {
		if symbol != "" && row.Symbol != symbol {
			continue
		}
		item := s.scoreCandidate(ctx, row, sectorScores[row.Industry])
		if item.Confidence < minConfidence {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		switch sortBy {
		case "technical":
			return items[i].TechnicalScore > items[j].TechnicalScore
		case "capital":
			return items[i].CapitalScore > items[j].CapitalScore
		case "risk":
			return items[i].RiskScore < items[j].RiskScore
		default:
			return items[i].TotalScore > items[j].TotalScore
		}
	})
	if len(items) > int(limit) {
		items = items[:limit]
	}
	snapshotNo := ""
	if persist && len(items) > 0 {
		if snapshotNo, err = s.persistCandidates(ctx, items); err != nil {
			return "", fmt.Errorf("persist candidate snapshot: %w", err)
		}
	}
	return envelope(map[string]any{"items": items, "count": len(items), "snapshot_no": snapshotNo, "snapshot_mode": map[bool]string{true: "REFRESHED", false: "LIVE_FALLBACK"}[persist], "factor_version": factorVersion, "scoring_version": scoringVersion, "source": "database", "data_mode": "real", "disclaimer": disclaimer})
}

func (s *Service) latestCandidateSnapshot(ctx context.Context, sortBy, industry string, minConfidence float64, limit int32, symbol string) (string, bool, error) {
	var snapshotNo string
	var snapshotTime time.Time
	err := s.db.QueryRowContext(ctx, `SELECT snapshot_no,MAX(created_at) FROM candidate_pool_snapshots GROUP BY snapshot_no ORDER BY MAX(created_at) DESC LIMIT 1`).Scan(&snapshotNo, &snapshotTime)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	query := `SELECT symbol,name,industry,total_score,technical_score,capital_score,sentiment_score,sector_score,risk_score,confidence,COALESCE(CAST(evidence_json AS CHAR),'{}'),factor_version,scoring_version,data_source,quote_time FROM candidate_pool_snapshots WHERE snapshot_no=?`
	args := []any{snapshotNo}
	if strings.TrimSpace(industry) != "" {
		query += ` AND industry=?`
		args = append(args, industry)
	}
	if strings.TrimSpace(symbol) != "" {
		query += ` AND symbol=?`
		args = append(args, symbol)
	}
	query += ` AND confidence>=?`
	args = append(args, minConfidence)
	switch sortBy {
	case "technical":
		query += ` ORDER BY technical_score DESC`
	case "capital":
		query += ` ORDER BY capital_score DESC`
	case "risk":
		query += ` ORDER BY risk_score ASC`
	default:
		query += ` ORDER BY total_score DESC`
	}
	query += ` LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return "", false, err
	}
	defer rows.Close()
	items := make([]candidate, 0, limit)
	for rows.Next() {
		var item candidate
		var evidenceJSON string
		var quoteTime sql.NullTime
		if err := rows.Scan(&item.Symbol, &item.Name, &item.Industry, &item.TotalScore, &item.TechnicalScore, &item.CapitalScore, &item.SentimentScore, &item.SectorScore, &item.RiskScore, &item.Confidence, &evidenceJSON, &item.FactorVersion, &item.ScoringVersion, &item.DataSource, &quoteTime); err != nil {
			return "", false, err
		}
		var evidence struct {
			Reasons         []string       `json:"reasons"`
			NegativeFactors []string       `json:"negative_factors"`
			Evidence        map[string]any `json:"evidence"`
		}
		_ = json.Unmarshal([]byte(evidenceJSON), &evidence)
		item.Reasons, item.NegativeFactors, item.Evidence = evidence.Reasons, evidence.NegativeFactors, evidence.Evidence
		item.DataMode, item.Disclaimer = "real", disclaimer
		if quoteTime.Valid {
			item.QuoteTime = quoteTime.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return "", false, err
	}
	raw, err := envelope(map[string]any{"items": items, "count": len(items), "snapshot_no": snapshotNo, "snapshot_time": snapshotTime.Format("2006-01-02 15:04:05"), "snapshot_mode": "CLOSE_SNAPSHOT", "factor_version": factorVersion, "scoring_version": scoringVersion, "source": "database", "data_mode": "real", "disclaimer": disclaimer})
	return raw, err == nil, err
}

func (s *Service) DragonTiger(ctx context.Context, tradeDate string, limit int32) (string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT DATE_FORMAT(trade_date,'%Y-%m-%d'),symbol,name,reason,close_price,change_percent,net_buy_amount,buy_amount,sell_amount,turnover_amount,source,collected_at FROM dragon_tiger_records`
	args := []any{}
	if tradeDate != "" {
		query += ` WHERE trade_date=?`
		args = append(args, tradeDate)
	}
	query += ` ORDER BY trade_date DESC,ABS(net_buy_amount) DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var date, symbol, name, reason, source string
		var close, change, net, buy, sell, turnover float64
		var collected time.Time
		if err = rows.Scan(&date, &symbol, &name, &reason, &close, &change, &net, &buy, &sell, &turnover, &source, &collected); err != nil {
			return "", err
		}
		items = append(items, map[string]any{"trade_date": date, "symbol": symbol, "name": name, "reason": reason, "close_price": close, "change_percent": normalizePct(change), "net_buy_amount": net, "buy_amount": buy, "sell_amount": sell, "turnover_amount": turnover, "source": source, "collected_at": collected.Format(time.RFC3339)})
	}
	status := "AVAILABLE"
	message := "数据已加载"
	if len(items) == 0 {
		status = "SOURCE_PENDING"
		message = "当前暂无可验证的龙虎榜数据，系统不会使用模拟数据填充"
	}
	return envelope(map[string]any{"items": items, "count": len(items), "status": status, "message": message, "data_mode": "real", "disclaimer": disclaimer})
}

func (s *Service) loadQuotes(ctx context.Context, industry string, limit int) ([]quoteRow, error) {
	query := `SELECT q.symbol,COALESCE(q.name,s.name,''),COALESCE(s.industry,'未分类'),COALESCE(q.latest_price,q.price,0),COALESCE(q.change_percent,q.change_pct,0),COALESCE(q.volume,0),COALESCE(q.amount,0),COALESCE(q.turnover_rate,0),COALESCE(q.amplitude,0),q.quote_time,COALESCE(q.source,'database'),COALESCE(q.data_mode,'real') FROM realtime_quotes q LEFT JOIN stocks s ON s.symbol=q.symbol AND s.deleted_at IS NULL WHERE COALESCE(q.latest_price,q.price,0)>0`
	args := []any{}
	if strings.TrimSpace(industry) != "" {
		query += ` AND s.industry=?`
		args = append(args, industry)
	}
	query += ` ORDER BY q.amount DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]quoteRow, 0)
	for rows.Next() {
		var r quoteRow
		if err = rows.Scan(&r.Symbol, &r.Name, &r.Industry, &r.Price, &r.Change, &r.Volume, &r.Amount, &r.Turnover, &r.Amplitude, &r.QuoteTime, &r.Source, &r.DataMode); err != nil {
			return nil, err
		}
		r.Change = normalizePct(r.Change)
		r.Turnover = normalizePct(r.Turnover)
		r.Amplitude = normalizePct(r.Amplitude)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) scoreCandidate(ctx context.Context, row quoteRow, sectorStrength float64) candidate {
	closes, volumes := s.history(ctx, row.Symbol, 60)
	coverage := math.Min(1, float64(len(closes))/40)
	technical := 50.0
	capital := 50.0
	sentiment := clamp(50+row.Change*500, 0, 100)
	risk := 50.0
	reasons := []string{}
	negatives := []string{}
	ma5, ma20 := avgTail(closes, 5), avgTail(closes, 20)
	ret20 := 0.0
	volatility := 0.0
	if len(closes) >= 20 {
		ret20 = closes[len(closes)-1]/closes[len(closes)-20] - 1
		technical = clamp(50+ret20*180, 0, 100)
		if row.Price > ma5 {
			technical += 8
			reasons = append(reasons, "现价位于5日均线上方")
		}
		if ma5 > ma20 {
			technical += 12
			reasons = append(reasons, "短期均线高于20日均线")
		}
		volatility = returnStd(closes) * math.Sqrt(252)
		risk = clamp(volatility*100, 0, 100)
	} else {
		negatives = append(negatives, "历史K线覆盖不足")
	}
	avgVolume := avgTail(volumes, 20)
	volumeRatio := 0.0
	if avgVolume > 0 {
		volumeRatio = row.Volume / avgVolume
	}
	capital = clamp(35+math.Log10(math.Max(row.Amount, 1))/9*35+math.Min(volumeRatio, 3)*10+math.Min(row.Turnover*100, 15), 0, 100)
	if volumeRatio >= 1.5 {
		reasons = append(reasons, "成交量高于近20日均量")
	}
	if row.Change < 0 {
		negatives = append(negatives, "当前涨跌幅为负")
	}
	if risk >= 70 {
		negatives = append(negatives, "历史波动率较高")
	}
	if strings.Contains(strings.ToUpper(row.Name), "ST") {
		risk = 100
		negatives = append(negatives, "股票名称包含ST风险标识")
	}
	technical = clamp(technical, 0, 100)
	sectorStrength = clamp(sectorStrength, 0, 100)
	total := technical*.32 + capital*.23 + sentiment*.15 + sectorStrength*.15 + (100-risk)*.15
	confidence := coverage
	if row.DataMode != "real" {
		confidence *= .7
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "量价与趋势指标处于中性区间")
	}
	return candidate{Symbol: row.Symbol, Name: row.Name, Industry: row.Industry, Price: row.Price, ChangePercent: row.Change, Amount: row.Amount, TurnoverRate: row.Turnover, TotalScore: round(total), TechnicalScore: round(technical), CapitalScore: round(capital), SentimentScore: round(sentiment), SectorScore: round(sectorStrength), RiskScore: round(risk), Confidence: round8(confidence), Reasons: reasons, NegativeFactors: negatives, Evidence: map[string]any{"ma5": round(ma5), "ma20": round(ma20), "return_20d": round8(ret20), "volume_ratio": round(volumeRatio), "annual_volatility": round8(volatility), "history_count": len(closes)}, FactorVersion: factorVersion, ScoringVersion: scoringVersion, QuoteTime: row.QuoteTime.Format(time.RFC3339), DataSource: row.Source, DataMode: row.DataMode, Disclaimer: disclaimer}
}

func (s *Service) history(ctx context.Context, symbol string, limit int) ([]float64, []float64) {
	rows, err := s.db.QueryContext(ctx, `SELECT close,volume FROM stock_daily_bars WHERE symbol=? ORDER BY trade_date DESC LIMIT ?`, symbol, limit)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	c, v := []float64{}, []float64{}
	for rows.Next() {
		var x, y float64
		if rows.Scan(&x, &y) == nil {
			c = append(c, x)
			v = append(v, y)
		}
	}
	reverse(c)
	reverse(v)
	return c, v
}
func (s *Service) persistCandidates(ctx context.Context, items []candidate) (string, error) {
	snapshot := fmt.Sprintf("CP%s", time.Now().Format("20060102150405.000"))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range items {
		e, _ := json.Marshal(map[string]any{"reasons": item.Reasons, "negative_factors": item.NegativeFactors, "evidence": item.Evidence})
		quoteTime, err := time.Parse(time.RFC3339, item.QuoteTime)
		if err != nil {
			quoteTime = time.Now()
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO candidate_pool_snapshots(snapshot_no,symbol,name,industry,total_score,technical_score,capital_score,sentiment_score,sector_score,risk_score,confidence,evidence_json,factor_version,scoring_version,data_source,quote_time) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, snapshot, item.Symbol, item.Name, item.Industry, item.TotalScore, item.TechnicalScore, item.CapitalScore, item.SentimentScore, item.SectorScore, item.RiskScore, item.Confidence, e, factorVersion, scoringVersion, item.DataSource, quoteTime)
		if err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return snapshot, nil
}

func calculateOverview(rows []quoteRow) overview {
	o := overview{Total: len(rows), Source: "database", DataMode: "real", Disclaimer: disclaimer}
	changes := make([]float64, 0, len(rows))
	for _, r := range rows {
		changes = append(changes, r.Change)
		o.TotalAmount += r.Amount
		if r.Change > .0001 {
			o.RiseCount++
		} else if r.Change < -0.0001 {
			o.FallCount++
		} else {
			o.FlatCount++
		}
		if r.Change >= .095 {
			o.LimitUpCount++
		}
		if r.Change <= -0.095 {
			o.LimitDownCount++
		}
		if !r.QuoteTime.IsZero() && r.QuoteTime.Format(time.RFC3339) > o.QuoteTime {
			o.QuoteTime = r.QuoteTime.Format(time.RFC3339)
		}
	}
	if len(rows) > 0 {
		sort.Float64s(changes)
		o.MedianChange = changes[len(changes)/2]
		o.AverageChange = mean(changes)
		o.RiseRatio = float64(o.RiseCount) / float64(len(rows))
		o.MarketTemperature = clamp(o.RiseRatio*70+clamp((o.AverageChange+.02)/.04*30, 0, 30), 0, 100)
	}
	return o
}
func calculateSectors(rows []quoteRow) []sector {
	groups := map[string][]quoteRow{}
	for _, r := range rows {
		groups[r.Industry] = append(groups[r.Industry], r)
	}
	out := make([]sector, 0, len(groups))
	for name, g := range groups {
		s := sector{Code: slug(name), Name: name, Type: "industry", StockCount: len(g), LeaderChange: -999, Status: "INSUFFICIENT_COVERAGE", Source: "stocks+realtime_quotes", DataMode: "unavailable", Methodology: "equal_weight"}
		latestDate := latestRealQuoteDate(g)
		for _, r := range g {
			if !sectorQuoteOnDate(r, latestDate) {
				continue
			}
			s.QuotedCount++
			s.TotalAmount += r.Amount
			if r.Change > 0 {
				s.RiseCount++
			} else if r.Change < 0 {
				s.FallCount++
			}
			s.AverageChange += r.Change
			if r.Change > s.LeaderChange {
				s.LeaderChange = r.Change
				s.LeaderSymbol = r.Symbol
				s.LeaderName = r.Name
			}
			if quoteTime := formatQuoteTime(r.QuoteTime); quoteTime > s.QuoteTime {
				s.QuoteTime = quoteTime
			}
		}
		if s.QuotedCount > 0 {
			s.DataMode = "real"
			s.AverageChange /= float64(s.QuotedCount)
			s.CoverageRate = float64(s.QuotedCount) / float64(s.StockCount)
			breadth := float64(s.RiseCount-s.FallCount) / float64(s.QuotedCount)
			s.StrengthScore = round(clamp(50+s.AverageChange*400+breadth*20, 0, 100))
			if s.CoverageRate >= .8 {
				today := markettime.Now().Format("2006-01-02")
				switch {
				case markettime.IsContinuousAuction(time.Now()) && latestDate == today:
					s.Status = "AVAILABLE"
				case !markettime.IsContinuousAuction(time.Now()):
					s.Status = "CLOSED_SNAPSHOT"
				default:
					s.Status = "STALE_SNAPSHOT"
				}
			}
		}
		if s.LeaderChange == -999 {
			s.LeaderChange = 0
		}
		out = append(out, s)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StrengthScore > out[j].StrengthScore })
	return out
}

func latestRealQuoteDate(rows []quoteRow) string {
	latest := ""
	for _, row := range rows {
		if strings.ToLower(row.DataMode) != "real" || row.QuoteTime.IsZero() {
			continue
		}
		if day := row.QuoteTime.In(markettime.Location()).Format("2006-01-02"); day > latest {
			latest = day
		}
	}
	return latest
}

func sectorQuoteOnDate(row quoteRow, day string) bool {
	return day != "" && strings.ToLower(row.DataMode) == "real" && !row.QuoteTime.IsZero() && row.QuoteTime.In(markettime.Location()).Format("2006-01-02") == day && hasQuote(row)
}

func hasQuote(row quoteRow) bool {
	return row.Price > 0 || row.Amount > 0 || !row.QuoteTime.IsZero()
}
func distribution(rows []quoteRow) []map[string]any {
	ranges := []struct {
		label    string
		min, max float64
	}{{"跌幅超过7%", -99, -.07}, {"跌幅3%至7%", -.07, -.03}, {"跌幅0至3%", -.03, 0}, {"上涨0至3%", 0, .03}, {"上涨3%至7%", .03, .07}, {"涨幅超过7%", .07, 99}}
	out := make([]map[string]any, 0, len(ranges))
	for _, rg := range ranges {
		n := 0
		for _, r := range rows {
			if r.Change >= rg.min && r.Change < rg.max {
				n++
			}
		}
		out = append(out, map[string]any{"label": rg.label, "count": n})
	}
	return out
}
func sectorScoreMap(items []sector) map[string]float64 {
	m := map[string]float64{}
	for _, x := range items {
		m[x.Name] = x.StrengthScore
	}
	return m
}
func sortQuotes(rows []quoteRow, sortBy, order string) {
	desc := normalizeOrder(order) == "desc"
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := quoteSortValue(rows[i], sortBy), quoteSortValue(rows[j], sortBy)
		if desc {
			return a > b
		}
		return a < b
	})
}
func quoteSortValue(r quoteRow, key string) float64 {
	switch normalizeSort(key) {
	case "amount":
		return r.Amount
	case "turnover_rate":
		return r.Turnover
	case "volume":
		return r.Volume
	default:
		return r.Change
	}
}
func normalizeSort(v string) string {
	switch v {
	case "amount", "turnover_rate", "volume", "change_percent":
		return v
	default:
		return "change_percent"
	}
}
func normalizeOrder(v string) string {
	if strings.EqualFold(v, "asc") {
		return "asc"
	}
	return "desc"
}
func normalizePct(v float64) float64 {
	if math.Abs(v) > 1 {
		return v / 100
	}
	return v
}
func avgTail(v []float64, n int) float64 {
	if len(v) == 0 {
		return 0
	}
	if len(v) < n {
		n = len(v)
	}
	return mean(v[len(v)-n:])
}
func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}
func returnStd(c []float64) float64 {
	if len(c) < 3 {
		return 0
	}
	r := make([]float64, 0, len(c)-1)
	for i := 1; i < len(c); i++ {
		if c[i-1] > 0 {
			r = append(r, c[i]/c[i-1]-1)
		}
	}
	m := mean(r)
	s := 0.0
	for _, x := range r {
		s += (x - m) * (x - m)
	}
	if len(r) < 2 {
		return 0
	}
	return math.Sqrt(s / float64(len(r)-1))
}
func reverse(v []float64) {
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
}
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func round(v float64) float64  { return math.Round(v*100) / 100 }
func round8(v float64) float64 { return math.Round(v*1e8) / 1e8 }
func slug(v string) string     { return fmt.Sprintf("SEC-%x", []byte(v)) }
func latestQuoteTime(rows []quoteRow) string {
	v := ""
	for _, r := range rows {
		t := formatQuoteTime(r.QuoteTime)
		if t > v {
			v = t
		}
	}
	return v
}
func formatQuoteTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
func envelope(data any) (string, error) {
	raw, err := json.Marshal(map[string]any{"code": 0, "message": "success", "data": data})
	return string(raw), err
}
