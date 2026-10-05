package domain

import (
	"encoding/json"
	"testing"

	"quant-system/backend-go/apps/strategy-rpc/strategy"
)

func TestNumericValue(t *testing.T) {
	tests := []struct {
		input any
		want  float64
	}{{72.5, 72.5}, {"68", 68}, {json.Number("45.25"), 45.25}, {int64(30), 30}}
	for _, tt := range tests {
		if got := numericValue(tt.input); got != tt.want {
			t.Fatalf("numericValue(%v)=%v want %v", tt.input, got, tt.want)
		}
	}
}

func TestEffectiveStrategyConfigUsesCustomBaseAndParams(t *testing.T) {
	base, params, err := effectiveStrategyConfig(&strategy.StrategyInfo{
		Code: "custom_1", ParamsJson: `{"base_strategy":"ma_cross","short_window":8,"long_window":34}`,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if base != "ma_cross" {
		t.Fatalf("base=%s", base)
	}
	if params != `{"long_window":34,"short_window":8}` {
		t.Fatalf("params=%s", params)
	}
}

func TestEffectiveStrategyConfigRejectsUnknownAlgorithm(t *testing.T) {
	_, _, err := effectiveStrategyConfig(&strategy.StrategyInfo{Code: "custom_1", ParamsJson: `{"base_strategy":"arbitrary_python"}`}, "")
	if err == nil {
		t.Fatal("unknown strategy algorithm must be rejected")
	}
}
