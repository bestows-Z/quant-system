package strategyparams

import (
	"fmt"
	"math"
)

type parameter struct {
	min, max, fallback float64
	integer            bool
}

var specs = map[string]map[string]parameter{
	"ma_cross": {
		"short_window": {2, 60, 5, true}, "long_window": {3, 250, 20, true},
	},
	"macd_cross": {
		"fast": {2, 60, 12, true}, "slow": {3, 120, 26, true}, "signal": {2, 60, 9, true},
	},
	"rsi_reversal": {
		"period": {2, 60, 14, true}, "low": {1, 49, 30, false}, "high": {51, 99, 70, false},
	},
	"boll_breakout": {
		"period": {5, 120, 20, true}, "std": {0.5, 5, 2, false},
	},
	"volume_breakout": {
		"volume_window": {5, 120, 20, true}, "volume_multiplier": {1.05, 10, 1.8, false},
	},
	"trend_score": {},
}

// Validate checks that every parameter is understood by the execution engine.
// It returns the canonical strategy code used by both backtests and live signals.
func Validate(code string, params map[string]any) (string, error) {
	switch code {
	case "moving_average_cross":
		code = "ma_cross"
	case "rsi_overbought_oversold", "rsi":
		code = "rsi_reversal"
	case "boll":
		code = "boll_breakout"
	}
	spec, ok := specs[code]
	if !ok {
		return "", fmt.Errorf("不支持的策略算法: %s", code)
	}
	if params == nil {
		params = map[string]any{}
	}
	values := make(map[string]float64, len(spec))
	for key, rule := range spec {
		values[key] = rule.fallback
	}
	for key, raw := range params {
		rule, exists := spec[key]
		if !exists {
			return "", fmt.Errorf("策略 %s 不支持参数 %s", code, key)
		}
		value, ok := number(raw)
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) || value < rule.min || value > rule.max || (rule.integer && math.Trunc(value) != value) {
			return "", fmt.Errorf("策略参数 %s 超出允许范围或类型错误", key)
		}
		values[key] = value
	}
	if code == "ma_cross" && values["long_window"] <= values["short_window"] {
		return "", fmt.Errorf("长期均线必须大于短期均线")
	}
	if code == "macd_cross" && values["slow"] <= values["fast"] {
		return "", fmt.Errorf("MACD 慢线周期必须大于快线周期")
	}
	return code, nil
}

func number(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	default:
		return 0, false
	}
}
