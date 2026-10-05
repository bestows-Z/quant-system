package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func parseJSON(text string) map[string]any {
	var root map[string]any
	if err := json.Unmarshal([]byte(text), &root); err != nil {
		return map[string]any{"raw": text}
	}
	return root
}

func dataMap(root map[string]any) map[string]any {
	if m, ok := root["data"].(map[string]any); ok {
		return m
	}
	return root
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	case string:
		clean := strings.TrimSpace(strings.TrimSuffix(x, "%"))
		f, _ := strconv.ParseFloat(clean, 64)
		return f
	default:
		return 0
	}
}

func numFrom(root map[string]any, names ...string) float64 {
	for _, name := range names {
		if v, ok := root[name]; ok {
			return asFloat(v)
		}
	}
	for _, value := range root {
		if m, ok := value.(map[string]any); ok {
			if f := numFrom(m, names...); f != 0 {
				return f
			}
		}
	}
	return 0
}

func rawJSON(parts map[string]any) string {
	body, err := json.Marshal(parts)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(body)
}

func parseMoney(v string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f
}

func pctAbs(v float64) float64 {
	return math.Abs(v)
}
