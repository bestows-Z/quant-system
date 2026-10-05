package strategyparams

import "testing"

func TestValidateRejectsIgnoredOrInvalidParameters(t *testing.T) {
	tests := []struct {
		name, code string
		params     map[string]any
	}{
		{"unknown key", "ma_cross", map[string]any{"fast_period": 5}},
		{"invalid relationship", "ma_cross", map[string]any{"short_window": 30, "long_window": 20}},
		{"fractional period", "rsi_reversal", map[string]any{"period": 14.5}},
		{"string number", "boll_breakout", map[string]any{"std": "2"}},
		{"unused trend parameter", "trend_score", map[string]any{"ma_fast": 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Validate(tt.code, tt.params); err == nil {
				t.Fatal("invalid parameters must be rejected")
			}
		})
	}
}

func TestValidateNormalizesAliases(t *testing.T) {
	code, err := Validate("rsi", map[string]any{"period": 14, "low": 30, "high": 70})
	if err != nil || code != "rsi_reversal" {
		t.Fatalf("alias validation failed: %s %v", code, err)
	}
	if _, err := Validate("ma_cross", map[string]any{"short_window": 8, "long_window": 34}); err != nil {
		t.Fatal(err)
	}
}
