package domain

import (
	"math"
	"testing"
)

func TestCalculateAdvancedProducesEnterpriseRiskMetrics(t *testing.T) {
	series := []historicalSeries{
		{Symbol: "600176", Name: "中国巨石", Industry: "材料", Weight: 0.6, Returns: map[string]float64{"2026-01-02": 0.01, "2026-01-05": -0.02, "2026-01-06": 0.015, "2026-01-07": -0.03}},
		{Symbol: "000001", Name: "平安银行", Industry: "金融", Weight: 0.4, Returns: map[string]float64{"2026-01-02": 0.005, "2026-01-05": -0.01, "2026-01-06": 0.008, "2026-01-07": -0.01}},
	}
	result := calculateAdvanced(series, 100000, []string{"local_csv"}, []string{"real"})
	if result.ObservationCount != 4 || len(result.EquityCurve) != 4 {
		t.Fatalf("unexpected observations: %+v", result)
	}
	if result.AnnualVolatility <= 0 || result.MaxDrawdown <= 0 || result.VaR95 <= 0 || result.CVaR95 < result.VaR95 {
		t.Fatalf("risk metrics not calculated: %+v", result)
	}
	if math.Abs(result.HHI-0.52) > 1e-8 || math.Abs(result.EffectivePositions-(1.0/0.52)) > 1e-6 {
		t.Fatalf("diversification metrics invalid: %+v", result)
	}
	if len(result.Contributions) != 2 || len(result.Correlations) != 2 || len(result.Scenarios) != 4 {
		t.Fatalf("analysis dimensions missing: %+v", result)
	}
	if result.Scenarios[0].LossAmount != 5000 {
		t.Fatalf("market scenario amount = %v", result.Scenarios[0].LossAmount)
	}
}

func TestCalculateAdvancedRequiresEightyPercentDailyCoverage(t *testing.T) {
	series := []historicalSeries{
		{Symbol: "A", Weight: 0.7, Returns: map[string]float64{"2026-01-02": 0.01, "2026-01-03": 0.01}},
		{Symbol: "B", Weight: 0.3, Returns: map[string]float64{"2026-01-02": 0.01}},
	}
	result := calculateAdvanced(series, 100, nil, nil)
	if result.ObservationCount != 1 || result.HistoryCoverage != 0.5 {
		t.Fatalf("coverage threshold not enforced: %+v", result)
	}
}

func TestMaxDrawdownIncludesFirstReturnFromInitialEquity(t *testing.T) {
	points := []equityPoint{{Date: "2026-01-02", Value: 0.9}, {Date: "2026-01-03", Value: 0.95}}
	if got := maxDrawdown(points); math.Abs(got-0.1) > 1e-10 {
		t.Fatalf("max drawdown = %v, want 0.1", got)
	}
}

func TestCorrelationRejectsInsufficientAndConstantSamples(t *testing.T) {
	if _, ok := correlation([]float64{0.1}, []float64{0.2}); ok {
		t.Fatal("single observation must not produce correlation")
	}
	if _, ok := correlation([]float64{0.1, 0.1}, []float64{0.2, 0.3}); ok {
		t.Fatal("constant series must not produce correlation")
	}
}
