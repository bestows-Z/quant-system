package domain

import (
	"math"
	"sort"
)

const annualTradingDays = 252.0

type historicalSeries struct {
	Symbol   string
	Name     string
	Industry string
	Weight   float64
	Returns  map[string]float64
}

type equityPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type contribution struct {
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
	Risk         float64 `json:"risk_contribution"`
	Return       float64 `json:"return_contribution"`
	PeriodReturn float64 `json:"period_return"`
}

type correlationRow struct {
	Symbol string              `json:"symbol"`
	Values map[string]*float64 `json:"values"`
}

type scenarioResult struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	LossRate    float64 `json:"loss_rate"`
	LossAmount  float64 `json:"loss_amount"`
	Level       string  `json:"level"`
}

type advancedAnalysis struct {
	ObservationCount   int              `json:"observation_count"`
	StartDate          string           `json:"start_date"`
	EndDate            string           `json:"end_date"`
	HistoryCoverage    float64          `json:"history_coverage"`
	AnnualVolatility   float64          `json:"annual_volatility"`
	AnnualReturn       float64          `json:"annual_return"`
	SharpeRatio        float64          `json:"sharpe_ratio"`
	SortinoRatio       float64          `json:"sortino_ratio"`
	MaxDrawdown        float64          `json:"max_drawdown"`
	VaR95              float64          `json:"var_95"`
	CVaR95             float64          `json:"cvar_95"`
	HHI                float64          `json:"hhi"`
	EffectivePositions float64          `json:"effective_positions"`
	EquityCurve        []equityPoint    `json:"equity_curve"`
	Contributions      []contribution   `json:"contributions"`
	Correlations       []correlationRow `json:"correlations"`
	Scenarios          []scenarioResult `json:"scenarios"`
	HistorySources     []string         `json:"history_sources"`
	HistoryModes       []string         `json:"history_modes"`
}

func calculateAdvanced(series []historicalSeries, marketValue float64, historySources, historyModes []string) advancedAnalysis {
	out := advancedAnalysis{
		EquityCurve:    make([]equityPoint, 0),
		Contributions:  make([]contribution, 0, len(series)),
		Correlations:   make([]correlationRow, 0, len(series)),
		Scenarios:      make([]scenarioResult, 0, 4),
		HistorySources: historySources,
		HistoryModes:   historyModes,
	}
	if len(series) == 0 {
		return out
	}

	weightSum, hhi := 0.0, 0.0
	for _, item := range series {
		weightSum += item.Weight
		hhi += item.Weight * item.Weight
	}
	out.HHI = finite(hhi)
	if hhi > 0 {
		out.EffectivePositions = finite(1 / hhi)
	}

	dates := commonDates(series)
	portfolioReturns := make([]float64, 0, len(dates))
	usedDates := make([]string, 0, len(dates))
	equity := 1.0
	for _, date := range dates {
		coveredWeight, daily := 0.0, 0.0
		for _, item := range series {
			if value, ok := item.Returns[date]; ok {
				coveredWeight += item.Weight
				daily += item.Weight * value
			}
		}
		if weightSum == 0 || coveredWeight/weightSum < 0.8 || coveredWeight == 0 {
			continue
		}
		daily /= coveredWeight
		portfolioReturns = append(portfolioReturns, daily)
		usedDates = append(usedDates, date)
		equity *= 1 + daily
		out.EquityCurve = append(out.EquityCurve, equityPoint{Date: date, Value: finite(equity)})
	}
	out.ObservationCount = len(portfolioReturns)
	if len(usedDates) > 0 {
		out.StartDate, out.EndDate = usedDates[0], usedDates[len(usedDates)-1]
	}
	if len(dates) > 0 {
		out.HistoryCoverage = float64(len(usedDates)) / float64(len(dates))
	}
	if len(portfolioReturns) >= 2 {
		meanReturn := mean(portfolioReturns)
		std := sampleStd(portfolioReturns)
		out.AnnualVolatility = finite(std * math.Sqrt(annualTradingDays))
		out.AnnualReturn = finite(math.Pow(equity, annualTradingDays/float64(len(portfolioReturns))) - 1)
		if std > 0 {
			out.SharpeRatio = finite(meanReturn / std * math.Sqrt(annualTradingDays))
		}
		downside := make([]float64, 0)
		for _, value := range portfolioReturns {
			if value < 0 {
				downside = append(downside, value)
			}
		}
		if downsideStd := rootMeanSquare(downside); downsideStd > 0 {
			out.SortinoRatio = finite(meanReturn / downsideStd * math.Sqrt(annualTradingDays))
		}
		out.MaxDrawdown = finite(maxDrawdown(out.EquityCurve))
		out.VaR95, out.CVaR95 = historicalTailRisk(portfolioReturns)
	}
	out.Contributions = calculateContributions(series, portfolioReturns, usedDates)
	out.Correlations = calculateCorrelations(series)
	out.Scenarios = calculateScenarios(series, marketValue)
	return out
}

func commonDates(series []historicalSeries) []string {
	seen := map[string]bool{}
	for _, item := range series {
		for date := range item.Returns {
			seen[date] = true
		}
	}
	dates := make([]string, 0, len(seen))
	for date := range seen {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	return dates
}

func calculateContributions(series []historicalSeries, portfolioReturns []float64, dates []string) []contribution {
	out := make([]contribution, 0, len(series))
	variance := covariance(portfolioReturns, portfolioReturns)
	for _, item := range series {
		aligned := make([]float64, 0, len(dates))
		periodEquity := 1.0
		returnContribution := 0.0
		for index, date := range dates {
			coveredWeight := 0.0
			for _, candidate := range series {
				if _, ok := candidate.Returns[date]; ok {
					coveredWeight += candidate.Weight
				}
			}
			value, present := item.Returns[date]
			effectiveContribution := 0.0
			if present && coveredWeight > 0 {
				effectiveContribution = item.Weight / coveredWeight * value
				periodEquity *= 1 + value
			}
			aligned = append(aligned, effectiveContribution)
			if index < len(portfolioReturns) {
				returnContribution += effectiveContribution
			}
		}
		risk := 0.0
		if variance > 0 && len(aligned) == len(portfolioReturns) {
			risk = item.Weight * covariance(aligned, portfolioReturns) / variance
		}
		periodReturn := periodEquity - 1
		out = append(out, contribution{Symbol: item.Symbol, Name: item.Name, Weight: item.Weight, Risk: finite(risk), Return: finite(returnContribution), PeriodReturn: finite(periodReturn)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Risk > out[j].Risk })
	return out
}

func calculateCorrelations(series []historicalSeries) []correlationRow {
	out := make([]correlationRow, 0, len(series))
	for _, left := range series {
		values := make(map[string]*float64, len(series))
		for _, right := range series {
			x, y := alignReturns(left.Returns, right.Returns)
			if value, ok := correlation(x, y); ok {
				v := finite(value)
				values[right.Symbol] = &v
			} else {
				values[right.Symbol] = nil
			}
		}
		out = append(out, correlationRow{Symbol: left.Symbol, Values: values})
	}
	return out
}

func calculateScenarios(series []historicalSeries, marketValue float64) []scenarioResult {
	maxWeight := 0.0
	industryWeights := map[string]float64{}
	weightedVolShock := 0.0
	for _, item := range series {
		if item.Weight > maxWeight {
			maxWeight = item.Weight
		}
		industryWeights[item.Industry] += item.Weight
		values := make([]float64, 0, len(item.Returns))
		for _, value := range item.Returns {
			values = append(values, value)
		}
		shock := math.Min(0.20, math.Max(0.08, sampleStd(values)*math.Sqrt(annualTradingDays)))
		weightedVolShock += item.Weight * shock
	}
	maxIndustry := 0.0
	for _, weight := range industryWeights {
		if weight > maxIndustry {
			maxIndustry = weight
		}
	}
	items := []scenarioResult{
		{Code: "market_down_5", Name: "市场普跌 5%", Description: "全部持仓同步下跌 5%", LossRate: 0.05},
		{Code: "volatility_shock", Name: "高波动冲击", Description: "按各持仓历史波动率施加 8% 至 20% 下行冲击", LossRate: weightedVolShock},
		{Code: "top_industry_down_8", Name: "最大行业下跌 8%", Description: "权重最高行业整体下跌 8%", LossRate: maxIndustry * 0.08},
		{Code: "largest_position_down_10", Name: "最大持仓下跌 10%", Description: "单票权重最高的持仓下跌 10%", LossRate: maxWeight * 0.10},
	}
	for i := range items {
		items[i].LossRate = finite(items[i].LossRate)
		items[i].LossAmount = finite(items[i].LossRate * marketValue)
		items[i].Level = riskLevelForLoss(items[i].LossRate)
	}
	return items
}

func alignReturns(left, right map[string]float64) ([]float64, []float64) {
	dates := make([]string, 0)
	for date := range left {
		if _, ok := right[date]; ok {
			dates = append(dates, date)
		}
	}
	sort.Strings(dates)
	x, y := make([]float64, 0, len(dates)), make([]float64, 0, len(dates))
	for _, date := range dates {
		x, y = append(x, left[date]), append(y, right[date])
	}
	return x, y
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func sampleStd(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	m := mean(values)
	total := 0.0
	for _, value := range values {
		delta := value - m
		total += delta * delta
	}
	return math.Sqrt(total / float64(len(values)-1))
}

func rootMeanSquare(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value * value
	}
	return math.Sqrt(total / float64(len(values)))
}

func covariance(left, right []float64) float64 {
	if len(left) < 2 || len(left) != len(right) {
		return 0
	}
	lm, rm := mean(left), mean(right)
	total := 0.0
	for i := range left {
		total += (left[i] - lm) * (right[i] - rm)
	}
	return total / float64(len(left)-1)
}

func correlation(left, right []float64) (float64, bool) {
	if len(left) < 2 || len(left) != len(right) {
		return 0, false
	}
	denominator := sampleStd(left) * sampleStd(right)
	if denominator == 0 {
		return 0, false
	}
	return covariance(left, right) / denominator, true
}

func maxDrawdown(points []equityPoint) float64 {
	peak, drawdown := 1.0, 0.0
	for _, point := range points {
		if point.Value > peak {
			peak = point.Value
		}
		if peak > 0 {
			current := (peak - point.Value) / peak
			if current > drawdown {
				drawdown = current
			}
		}
	}
	return drawdown
}

func historicalTailRisk(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	index := int(math.Floor(float64(len(sorted)-1) * 0.05))
	threshold := sorted[index]
	tail := sorted[:index+1]
	return finite(math.Max(0, -threshold)), finite(math.Max(0, -mean(tail)))
}

func riskLevelForLoss(loss float64) string {
	if loss >= 0.15 {
		return "EXTREME"
	}
	if loss >= 0.08 {
		return "HIGH"
	}
	if loss >= 0.04 {
		return "MEDIUM"
	}
	return "LOW"
}

func finite(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*1e8) / 1e8
}
