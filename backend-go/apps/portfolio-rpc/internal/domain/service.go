package domain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	analysisVersion = "portfolio-v2"
	disclaimer      = "真实持仓数据分析，仅供研究，不构成投资建议；系统不连接券商或执行真实交易"
)

type Service struct {
	db     *sql.DB
	market marketclient.Market
}

type importPreview struct {
	Rows   []*portfolio.ImportRow   `json:"rows"`
	Errors []*portfolio.ImportError `json:"errors"`
}

type quote struct {
	Price     decimal.Decimal
	Name      string
	Source    string
	DataMode  string
	QuoteTime string
}

func NewService(db *sql.DB, market marketclient.Market) *Service {
	return &Service{db: db, market: market}
}

func (s *Service) CreatePortfolio(ctx context.Context, in *portfolio.CreatePortfolioReq) (*portfolio.Portfolio, error) {
	name := strings.TrimSpace(in.GetName())
	if in.GetUserId() == 0 || name == "" {
		return nil, status.Error(codes.InvalidArgument, "组合名称不能为空")
	}
	if len([]rune(name)) > 80 {
		return nil, status.Error(codes.InvalidArgument, "组合名称不能超过 80 个字符")
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO real_portfolios(user_id,name,description) VALUES(?,?,?)`, in.GetUserId(), name, strings.TrimSpace(in.GetDescription()))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, status.Error(codes.AlreadyExists, "同名组合已存在")
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetPortfolio(ctx, in.GetUserId(), uint64(id))
}

func (s *Service) ListPortfolios(ctx context.Context, userID uint64) ([]*portfolio.Portfolio, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,description,base_currency,status,version,created_at,updated_at FROM real_portfolios WHERE user_id=? AND deleted_at IS NULL ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*portfolio.Portfolio, 0)
	for rows.Next() {
		p, err := scanPortfolio(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (s *Service) GetPortfolio(ctx context.Context, userID, portfolioID uint64) (*portfolio.Portfolio, error) {
	p, err := scanPortfolio(s.db.QueryRowContext(ctx, `SELECT id,name,description,base_currency,status,version,created_at,updated_at FROM real_portfolios WHERE user_id=? AND id=? AND deleted_at IS NULL`, userID, portfolioID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "持仓组合不存在")
	}
	return p, err
}

func (s *Service) DeletePortfolio(ctx context.Context, userID, portfolioID uint64) error {
	if _, err := s.GetPortfolio(ctx, userID, portfolioID); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE real_positions SET deleted_at=CURRENT_TIMESTAMP(3),version=version+1 WHERE user_id=? AND portfolio_id=? AND deleted_at IS NULL`, userID, portfolioID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE real_portfolios SET deleted_at=CURRENT_TIMESTAMP(3),status='DELETED',version=version+1 WHERE user_id=? AND id=? AND deleted_at IS NULL`, userID, portfolioID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return status.Error(codes.NotFound, "持仓组合不存在")
	}
	return tx.Commit()
}

func (s *Service) UpsertPosition(ctx context.Context, in *portfolio.UpsertPositionReq) (*portfolio.Position, error) {
	if _, err := s.GetPortfolio(ctx, in.GetUserId(), in.GetPortfolioId()); err != nil {
		return nil, err
	}
	row, err := normalizePosition(in.GetSymbol(), in.GetName(), in.GetQuantity(), in.GetAvailableQuantity(), in.GetAvgCost(), in.GetOpenedOn(), in.GetNote(), 0)
	if err != nil {
		return nil, err
	}
	if in.GetPositionId() == 0 {
		res, err := s.db.ExecContext(ctx, `INSERT INTO real_positions(portfolio_id,user_id,symbol,name,quantity,available_quantity,avg_cost,opened_on,note) VALUES(?,?,?,?,?,?,?,NULLIF(?,''),?)`, in.GetPortfolioId(), in.GetUserId(), row.Symbol, row.Name, row.Quantity, row.AvailableQuantity, row.AvgCost, row.OpenedOn, row.Note)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return nil, status.Error(codes.AlreadyExists, "该股票已在组合中")
			}
			return nil, err
		}
		id, _ := res.LastInsertId()
		return s.getPosition(ctx, in.GetUserId(), in.GetPortfolioId(), uint64(id))
	}
	if in.GetVersion() == 0 {
		return nil, status.Error(codes.InvalidArgument, "更新持仓必须提供 version")
	}
	res, err := s.db.ExecContext(ctx, `UPDATE real_positions SET symbol=?,name=?,quantity=?,available_quantity=?,avg_cost=?,opened_on=NULLIF(?,''),note=?,version=version+1 WHERE id=? AND portfolio_id=? AND user_id=? AND version=? AND deleted_at IS NULL`, row.Symbol, row.Name, row.Quantity, row.AvailableQuantity, row.AvgCost, row.OpenedOn, row.Note, in.GetPositionId(), in.GetPortfolioId(), in.GetUserId(), in.GetVersion())
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, status.Error(codes.Aborted, "持仓已被修改，请刷新后重试")
	}
	return s.getPosition(ctx, in.GetUserId(), in.GetPortfolioId(), in.GetPositionId())
}

func (s *Service) DeletePosition(ctx context.Context, userID, portfolioID, positionID uint64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE real_positions SET deleted_at=CURRENT_TIMESTAMP(3),version=version+1 WHERE id=? AND portfolio_id=? AND user_id=? AND deleted_at IS NULL`, positionID, portfolioID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return status.Error(codes.NotFound, "持仓不存在")
	}
	return nil
}

func (s *Service) ListPositions(ctx context.Context, userID, portfolioID uint64) ([]*portfolio.Position, error) {
	if _, err := s.GetPortfolio(ctx, userID, portfolioID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, positionSelect+` WHERE user_id=? AND portfolio_id=? AND deleted_at IS NULL ORDER BY id DESC`, userID, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*portfolio.Position, 0)
	for rows.Next() {
		p, err := scanPosition(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (s *Service) PreviewImport(ctx context.Context, in *portfolio.PreviewImportReq) (*portfolio.PreviewImportResp, error) {
	if _, err := s.GetPortfolio(ctx, in.GetUserId(), in.GetPortfolioId()); err != nil {
		return nil, err
	}
	if len(in.GetCsvContent()) == 0 || len(in.GetCsvContent()) > 2*1024*1024 {
		return nil, status.Error(codes.InvalidArgument, "CSV 文件不能为空且不能超过 2MB")
	}
	key := strings.TrimSpace(in.GetIdempotencyKey())
	if key == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key 不能为空")
	}
	h := sha256.Sum256(in.GetCsvContent())
	hash := hex.EncodeToString(h[:])
	if existing, err := s.previewByKey(ctx, in.GetUserId(), in.GetPortfolioId(), key); err == nil {
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	rows, errs := ParseCSV(in.GetCsvContent())
	payload, _ := json.Marshal(importPreview{Rows: rows, Errors: errs})
	errJSON, _ := json.Marshal(errs)
	res, err := s.db.ExecContext(ctx, `INSERT INTO position_import_tasks(user_id,portfolio_id,file_name,file_hash,idempotency_key,status,total_rows,valid_rows,invalid_rows,preview_json,error_json) VALUES(?,?,?,?,?,'PREVIEWED',?,?,?,?,?)`, in.GetUserId(), in.GetPortfolioId(), safeFileName(in.GetFileName()), hash, key, len(rows)+len(errs), len(rows), len(errs), payload, errJSON)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &portfolio.PreviewImportResp{TaskId: uint64(id), FileHash: hash, Status: "PREVIEWED", TotalRows: uint32(len(rows) + len(errs)), ValidRows: uint32(len(rows)), InvalidRows: uint32(len(errs)), Rows: rows, Errors: errs}, nil
}

func (s *Service) ConfirmImport(ctx context.Context, userID, portfolioID, taskID uint64) (*portfolio.ConfirmImportResp, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var statusText string
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT status,preview_json FROM position_import_tasks WHERE id=? AND user_id=? AND portfolio_id=? FOR UPDATE`, taskID, userID, portfolioID).Scan(&statusText, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "导入任务不存在")
	}
	if err != nil {
		return nil, err
	}
	if statusText == "CONFIRMED" {
		return &portfolio.ConfirmImportResp{TaskId: taskID, Status: statusText, IdempotentReplay: true}, nil
	}
	var preview importPreview
	if err = json.Unmarshal(raw, &preview); err != nil {
		return nil, err
	}
	if len(preview.Errors) > 0 {
		return nil, status.Error(codes.FailedPrecondition, "预览包含错误行，不能确认导入")
	}
	for _, row := range preview.Rows {
		_, err = tx.ExecContext(ctx, `INSERT INTO real_positions(portfolio_id,user_id,symbol,name,quantity,available_quantity,avg_cost,opened_on,note) VALUES(?,?,?,?,?,?,?,NULLIF(?,''),?) ON DUPLICATE KEY UPDATE name=VALUES(name),quantity=VALUES(quantity),available_quantity=VALUES(available_quantity),avg_cost=VALUES(avg_cost),opened_on=VALUES(opened_on),note=VALUES(note),version=version+1,updated_at=CURRENT_TIMESTAMP(3)`, portfolioID, userID, row.Symbol, row.Name, row.Quantity, row.AvailableQuantity, row.AvgCost, row.OpenedOn, row.Note)
		if err != nil {
			return nil, err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE position_import_tasks SET status='CONFIRMED',confirmed_at=CURRENT_TIMESTAMP(3) WHERE id=?`, taskID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &portfolio.ConfirmImportResp{TaskId: taskID, Status: "CONFIRMED", ImportedRows: uint32(len(preview.Rows))}, nil
}

func (s *Service) Analyze(ctx context.Context, userID, portfolioID uint64) (*portfolio.PortfolioAnalysis, error) {
	positions, err := s.ListPositions(ctx, userID, portfolioID)
	if err != nil {
		return nil, err
	}
	result := &portfolio.PortfolioAnalysis{PortfolioId: portfolioID, AnalysisVersion: analysisVersion, Disclaimer: disclaimer, Positions: make([]*portfolio.PositionAnalysis, 0), IndustryExposures: make([]*portfolio.IndustryExposure, 0), Warnings: make([]string, 0)}
	if len(positions) == 0 {
		result.RiskLevel = "LOW"
		result.CoverageRate = "0"
		result.Warnings = append(result.Warnings, "组合暂无持仓")
		return result, nil
	}
	totalCost, coveredCost, totalMarket := decimal.Zero, decimal.Zero, decimal.Zero
	covered := 0
	latestTime := ""
	sourceSet := map[string]bool{}
	modeSet := map[string]bool{}
	type calc struct {
		pb       *portfolio.PositionAnalysis
		market   decimal.Decimal
		industry string
	}
	calculated := make([]calc, 0, len(positions))
	industryTotals := map[string]decimal.Decimal{}
	for _, p := range positions {
		cost, _ := decimal.NewFromString(p.AvgCost)
		costValue := cost.Mul(decimal.NewFromInt(p.Quantity))
		totalCost = totalCost.Add(costValue)
		q, qerr := s.latestQuote(ctx, p.Symbol)
		pa := &portfolio.PositionAnalysis{PositionId: p.Id, Symbol: p.Symbol, Name: p.Name, Quantity: p.Quantity, AvgCost: p.AvgCost, CostValue: costValue.StringFixed(2), Warnings: make([]string, 0)}
		industry := s.stockIndustry(ctx, p.Symbol)
		pa.Industry = industry
		mv := decimal.Zero
		if qerr != nil {
			pa.RiskLevel = "HIGH"
			pa.Warnings = append(pa.Warnings, "实时行情不可用，未计入当前市值")
		} else {
			covered++
			coveredCost = coveredCost.Add(costValue)
			mv = q.Price.Mul(decimal.NewFromInt(p.Quantity))
			profit := mv.Sub(costValue)
			rate := decimal.Zero
			if !costValue.IsZero() {
				rate = profit.Div(costValue)
			}
			pa.MarketPrice = q.Price.StringFixed(4)
			pa.MarketValue = mv.StringFixed(2)
			pa.Profit = profit.StringFixed(2)
			pa.ProfitRate = rate.StringFixed(6)
			pa.RiskLevel = positionRisk(rate)
			totalMarket = totalMarket.Add(mv)
			industryTotals[industry] = industryTotals[industry].Add(mv)
			sourceSet[q.Source] = true
			modeSet[q.DataMode] = true
			if q.QuoteTime > latestTime {
				latestTime = q.QuoteTime
			}
		}
		calculated = append(calculated, calc{pb: pa, market: mv, industry: industry})
	}
	maxWeight := decimal.Zero
	for _, c := range calculated {
		w := decimal.Zero
		if totalMarket.GreaterThan(decimal.Zero) {
			w = c.market.Div(totalMarket)
		}
		c.pb.Weight = w.StringFixed(6)
		if w.GreaterThan(maxWeight) {
			maxWeight = w
		}
		result.Positions = append(result.Positions, c.pb)
	}
	for industry, mv := range industryTotals {
		w := decimal.Zero
		if totalMarket.GreaterThan(decimal.Zero) {
			w = mv.Div(totalMarket)
		}
		result.IndustryExposures = append(result.IndustryExposures, &portfolio.IndustryExposure{Industry: industry, MarketValue: mv.StringFixed(2), Weight: w.StringFixed(6)})
	}
	profit := totalMarket.Sub(coveredCost)
	rate := decimal.Zero
	if !coveredCost.IsZero() {
		rate = profit.Div(coveredCost)
	}
	coverage := decimal.NewFromInt(int64(covered)).Div(decimal.NewFromInt(int64(len(positions))))
	result.QuoteTime = latestTime
	result.CostValue = totalCost.StringFixed(2)
	result.MarketValue = totalMarket.StringFixed(2)
	result.TotalProfit = profit.StringFixed(2)
	result.TotalProfitRate = rate.StringFixed(6)
	result.Concentration = maxWeight.StringFixed(6)
	result.CoverageRate = coverage.StringFixed(6)
	result.DataSource = joinKeys(sourceSet)
	result.DataMode = joinKeys(modeSet)
	result.RiskLevel = portfolioRisk(maxWeight, coverage)
	if coverage.LessThan(decimal.NewFromFloat(0.8)) {
		result.Warnings = append(result.Warnings, "行情覆盖率低于 80%，组合盈亏仅按已覆盖持仓计算")
	}
	if maxWeight.GreaterThan(decimal.NewFromFloat(0.5)) {
		result.Warnings = append(result.Warnings, "单票集中度超过 50%")
	}
	advanced := s.historicalAnalysis(ctx, result.Positions, parseFloat(result.MarketValue))
	advancedJSON, err := json.Marshal(advanced)
	if err != nil {
		return nil, err
	}
	result.AdvancedJson = string(advancedJSON)
	result.Volatility = decimal.NewFromFloat(advanced.AnnualVolatility).StringFixed(8)
	result.MaxDrawdown = decimal.NewFromFloat(advanced.MaxDrawdown).StringFixed(8)
	if advanced.ObservationCount < 30 {
		result.Warnings = append(result.Warnings, "有效历史观察少于 30 个交易日，统计指标仅供参考")
	}
	if advanced.HistoryCoverage < 0.8 {
		result.Warnings = append(result.Warnings, "历史行情覆盖率低于 80%，风险统计置信度已降低")
	}
	if err := s.saveAnalysis(ctx, userID, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) LatestAnalysis(ctx context.Context, userID, portfolioID uint64) (*portfolio.PortfolioAnalysis, error) {
	if _, err := s.GetPortfolio(ctx, userID, portfolioID); err != nil {
		return nil, err
	}
	var result portfolio.PortfolioAnalysis
	var quoteTime sql.NullTime
	var industryRaw, positionsRaw, warningsRaw []byte
	err := s.db.QueryRowContext(ctx, `SELECT analysis_version,quote_time,market_value,total_profit,total_profit_rate,concentration,volatility,max_drawdown,risk_level,coverage_rate,industry_exposure_json,position_analysis_json,warnings_json,COALESCE(advanced_json,JSON_OBJECT()),data_source,data_mode FROM portfolio_analysis_records WHERE user_id=? AND portfolio_id=? ORDER BY id DESC LIMIT 1`, userID, portfolioID).Scan(&result.AnalysisVersion, &quoteTime, &result.MarketValue, &result.TotalProfit, &result.TotalProfitRate, &result.Concentration, &result.Volatility, &result.MaxDrawdown, &result.RiskLevel, &result.CoverageRate, &industryRaw, &positionsRaw, &warningsRaw, &result.AdvancedJson, &result.DataSource, &result.DataMode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "尚无组合分析，请先执行重新分析")
	}
	if err != nil {
		return nil, err
	}
	result.PortfolioId = portfolioID
	result.Disclaimer = disclaimer
	if quoteTime.Valid {
		result.QuoteTime = quoteTime.Time.Format(time.RFC3339)
	}
	if err = json.Unmarshal(industryRaw, &result.IndustryExposures); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(positionsRaw, &result.Positions); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(warningsRaw, &result.Warnings); err != nil {
		return nil, err
	}
	totalCost := decimal.Zero
	for _, position := range result.Positions {
		value, _ := decimal.NewFromString(position.CostValue)
		totalCost = totalCost.Add(value)
	}
	result.CostValue = totalCost.StringFixed(2)
	return &result, nil
}

func ParseCSV(content []byte) ([]*portfolio.ImportRow, []*portfolio.ImportError) {
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})))
	r.TrimLeadingSpace = true
	header, err := r.Read()
	if err != nil {
		return nil, []*portfolio.ImportError{{RowNumber: 1, Field: "file", Message: "无法读取 CSV 表头"}}
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[normalizeHeader(h)] = i
	}
	for _, required := range []string{"symbol", "quantity", "avg_cost"} {
		if _, ok := idx[required]; !ok {
			return nil, []*portfolio.ImportError{{RowNumber: 1, Field: required, Message: "缺少必填列"}}
		}
	}
	rows := make([]*portfolio.ImportRow, 0)
	errs := make([]*portfolio.ImportError, 0)
	seen := map[string]bool{}
	line := 1
	for {
		record, e := r.Read()
		if errors.Is(e, io.EOF) {
			break
		}
		line++
		if e != nil {
			errs = append(errs, &portfolio.ImportError{RowNumber: uint32(line), Field: "row", Message: "CSV 行格式错误"})
			continue
		}
		get := func(k string) string {
			i, ok := idx[k]
			if !ok || i >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[i])
		}
		qty, _ := strconv.ParseInt(get("quantity"), 10, 64)
		availText := get("available_quantity")
		avail := qty
		if availText != "" {
			avail, _ = strconv.ParseInt(availText, 10, 64)
		}
		row, nerr := normalizePosition(get("symbol"), get("name"), qty, avail, get("avg_cost"), get("opened_on"), get("note"), uint32(line))
		if nerr != nil {
			errs = append(errs, &portfolio.ImportError{RowNumber: uint32(line), Field: "row", Message: status.Convert(nerr).Message()})
			continue
		}
		if seen[row.Symbol] {
			errs = append(errs, &portfolio.ImportError{RowNumber: uint32(line), Field: "symbol", Message: "文件内股票代码重复"})
			continue
		}
		seen[row.Symbol] = true
		rows = append(rows, row)
	}
	return rows, errs
}

func normalizePosition(symbol, name string, qty, avail int64, cost, opened, note string, row uint32) (*portfolio.ImportRow, error) {
	symbol = strings.TrimSpace(symbol)
	if len(symbol) != 6 {
		return nil, status.Error(codes.InvalidArgument, "股票代码必须为 6 位")
	}
	for _, c := range symbol {
		if c < '0' || c > '9' {
			return nil, status.Error(codes.InvalidArgument, "股票代码必须为数字")
		}
	}
	if qty <= 0 {
		return nil, status.Error(codes.InvalidArgument, "持仓数量必须大于 0")
	}
	if avail < 0 || avail > qty {
		return nil, status.Error(codes.InvalidArgument, "可用数量必须在 0 到持仓数量之间")
	}
	d, err := decimal.NewFromString(strings.TrimSpace(cost))
	if err != nil || !d.GreaterThan(decimal.Zero) {
		return nil, status.Error(codes.InvalidArgument, "成本价必须大于 0")
	}
	opened = strings.TrimSpace(opened)
	if opened != "" {
		if _, err = time.Parse("2006-01-02", opened); err != nil {
			return nil, status.Error(codes.InvalidArgument, "建仓日期格式必须为 YYYY-MM-DD")
		}
	}
	return &portfolio.ImportRow{RowNumber: row, Symbol: symbol, Name: strings.TrimSpace(name), Quantity: qty, AvailableQuantity: avail, AvgCost: d.StringFixed(6), OpenedOn: opened, Note: strings.TrimSpace(note)}, nil
}

type scanner interface{ Scan(...any) error }

func scanPortfolio(s scanner) (*portfolio.Portfolio, error) {
	var p portfolio.Portfolio
	var created, updated time.Time
	err := s.Scan(&p.Id, &p.Name, &p.Description, &p.BaseCurrency, &p.Status, &p.Version, &created, &updated)
	if err != nil {
		return nil, err
	}
	p.CreatedAt = created.Format(time.RFC3339)
	p.UpdatedAt = updated.Format(time.RFC3339)
	return &p, nil
}

const positionSelect = `SELECT id,portfolio_id,symbol,name,quantity,available_quantity,avg_cost,COALESCE(DATE_FORMAT(opened_on,'%Y-%m-%d'),''),note,version,created_at,updated_at FROM real_positions`

func scanPosition(s scanner) (*portfolio.Position, error) {
	var p portfolio.Position
	var created, updated time.Time
	err := s.Scan(&p.Id, &p.PortfolioId, &p.Symbol, &p.Name, &p.Quantity, &p.AvailableQuantity, &p.AvgCost, &p.OpenedOn, &p.Note, &p.Version, &created, &updated)
	if err != nil {
		return nil, err
	}
	p.CreatedAt = created.Format(time.RFC3339)
	p.UpdatedAt = updated.Format(time.RFC3339)
	return &p, nil
}
func (s *Service) getPosition(ctx context.Context, userID, portfolioID, positionID uint64) (*portfolio.Position, error) {
	p, err := scanPosition(s.db.QueryRowContext(ctx, positionSelect+` WHERE id=? AND portfolio_id=? AND user_id=? AND deleted_at IS NULL`, positionID, portfolioID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "持仓不存在")
	}
	return p, err
}
func (s *Service) previewByKey(ctx context.Context, userID, portfolioID uint64, key string) (*portfolio.PreviewImportResp, error) {
	var id uint64
	var hash, statusText string
	var total, valid, invalid uint32
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT id,file_hash,status,total_rows,valid_rows,invalid_rows,preview_json FROM position_import_tasks WHERE user_id=? AND portfolio_id=? AND idempotency_key=?`, userID, portfolioID, key).Scan(&id, &hash, &statusText, &total, &valid, &invalid, &raw)
	if err != nil {
		return nil, err
	}
	var p importPreview
	if err = json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &portfolio.PreviewImportResp{TaskId: id, FileHash: hash, Status: statusText, TotalRows: total, ValidRows: valid, InvalidRows: invalid, Rows: p.Rows, Errors: p.Errors}, nil
}
func (s *Service) latestQuote(ctx context.Context, symbol string) (quote, error) {
	reply, err := s.market.RealtimeBySymbol(ctx, &marketclient.SymbolRequest{Symbol: symbol})
	if err != nil {
		return quote{}, err
	}
	return parseQuoteRaw(reply.Json)
}

func parseQuoteRaw(rawText string) (quote, error) {
	type quotePayload struct {
		Price     json.RawMessage `json:"price"`
		Current   json.RawMessage `json:"current"`
		Name      string          `json:"name"`
		Source    string          `json:"source"`
		DataMode  string          `json:"data_mode"`
		QuoteTime string          `json:"quote_time"`
	}
	var body struct {
		Data struct {
			Items []quotePayload `json:"items"`
			quotePayload
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(rawText), &body); err != nil {
		return quote{}, err
	}
	q := body.Data.quotePayload
	if len(body.Data.Items) > 0 {
		q = body.Data.Items[0]
	}
	priceRaw := q.Price
	if len(priceRaw) == 0 {
		priceRaw = q.Current
	}
	var value any
	if err := json.Unmarshal(priceRaw, &value); err != nil {
		return quote{}, err
	}
	d, err := decimal.NewFromString(fmt.Sprint(value))
	if err != nil || !d.GreaterThan(decimal.Zero) {
		return quote{}, errors.New("invalid quote")
	}
	return quote{Price: d, Name: q.Name, Source: q.Source, DataMode: q.DataMode, QuoteTime: q.QuoteTime}, nil
}
func (s *Service) stockIndustry(ctx context.Context, symbol string) string {
	var v string
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(industry,'未分类') FROM stocks WHERE symbol=? AND deleted_at IS NULL ORDER BY id LIMIT 1`, symbol).Scan(&v); err != nil || strings.TrimSpace(v) == "" {
		return "未分类"
	}
	return v
}
func (s *Service) saveAnalysis(ctx context.Context, userID uint64, a *portfolio.PortfolioAnalysis) error {
	industry, _ := json.Marshal(a.IndustryExposures)
	positions, _ := json.Marshal(a.Positions)
	warnings, _ := json.Marshal(a.Warnings)
	_, err := s.db.ExecContext(ctx, `INSERT INTO portfolio_analysis_records(user_id,portfolio_id,analysis_version,quote_time,market_value,total_profit,total_profit_rate,concentration,volatility,max_drawdown,risk_level,coverage_rate,industry_exposure_json,position_analysis_json,warnings_json,advanced_json,data_source,data_mode) VALUES(?,?,?,NULLIF(?,''),?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, userID, a.PortfolioId, a.AnalysisVersion, a.QuoteTime, a.MarketValue, a.TotalProfit, a.TotalProfitRate, a.Concentration, a.Volatility, a.MaxDrawdown, a.RiskLevel, a.CoverageRate, industry, positions, warnings, nullJSON(a.AdvancedJson), a.DataSource, a.DataMode)
	return err
}
func nullJSON(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
func normalizeHeader(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	aliases := map[string]string{"股票代码": "symbol", "代码": "symbol", "股票名称": "name", "名称": "name", "持仓数量": "quantity", "数量": "quantity", "可用数量": "available_quantity", "成本价": "avg_cost", "建仓日期": "opened_on", "备注": "note"}
	if a, ok := aliases[v]; ok {
		return a
	}
	return v
}
func safeFileName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "positions.csv"
	}
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "\\", "_")
	if len(v) > 255 {
		return v[:255]
	}
	return v
}
func joinKeys(m map[string]bool) string {
	items := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			items = append(items, k)
		}
	}
	return strings.Join(items, ",")
}
func positionRisk(rate decimal.Decimal) string {
	if rate.LessThan(decimal.NewFromFloat(-0.2)) {
		return "HIGH"
	}
	if rate.LessThan(decimal.NewFromFloat(-0.1)) {
		return "MEDIUM"
	}
	return "LOW"
}
func portfolioRisk(concentration, coverage decimal.Decimal) string {
	if coverage.LessThan(decimal.NewFromFloat(0.5)) || concentration.GreaterThan(decimal.NewFromFloat(0.7)) {
		return "HIGH"
	}
	if coverage.LessThan(decimal.NewFromFloat(0.8)) || concentration.GreaterThan(decimal.NewFromFloat(0.5)) {
		return "MEDIUM"
	}
	return "LOW"
}
