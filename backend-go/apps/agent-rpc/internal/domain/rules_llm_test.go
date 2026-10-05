package domain

import (
	"strings"
	"testing"
)

func TestDigestDataRemovesInternalAndDuplicateFields(t *testing.T) {
	data := map[string]any{
		"code":    0,
		"message": "success",
		"quote": map[string]any{
			"symbol":         "600176",
			"latest_price":   18.6,
			"change_percent": 1.2,
			"source":         "tencent",
			"raw_data_json":  `{"secret":"raw"}`,
		},
		"kline": map[string]any{
			"symbol": "600176",
			"source": "database",
			"items":  []any{map[string]any{"trade_date": "2026-07-17", "close": 18.6}},
		},
	}

	digest := digestData(data, []string{"symbol", "price", "latest_price", "change_pct", "change_percent"}, 12)
	for _, forbidden := range []string{"success", "raw_data_json", "secret"} {
		if strings.Contains(digest, forbidden) {
			t.Fatalf("digest contains internal field %q: %s", forbidden, digest)
		}
	}
	if strings.Count(digest, "symbol=") != 1 || strings.Count(digest, "数据来源=") != 1 || strings.Count(digest, "价格=") != 1 || strings.Count(digest, "涨跌幅=") != 1 {
		t.Fatalf("digest contains duplicate semantic fields: %s", digest)
	}
}

func TestDisplayDigestValueLocalizesStatusSignalAndTime(t *testing.T) {
	cases := map[string]string{
		displayDigestValue("signal_type", "SELL"):               "卖出",
		displayDigestValue("status", "OPEN"):                    "待处理",
		displayDigestValue("quote_time", "2026-07-18T12:46:17"): "2026-07-18 12:46:17",
	}
	for actual, expected := range cases {
		if actual != expected {
			t.Fatalf("unexpected localized value %q, want %q", actual, expected)
		}
	}
}
