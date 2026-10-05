package domain

import (
	"math"
	"testing"
)

func TestParseKlineReturnsSupportsTradeDateAndStringClose(t *testing.T) {
	raw := `{"code":0,"data":{"source":"local_csv","data_mode":"real","items":[{"trade_date":"2026-01-02","close":"10.00"},{"trade_date":"2026-01-05","close":10.5},{"trade_date":"2026-01-06","close":"10.29"}]}}`
	returns, source, mode, err := parseKlineReturns(raw)
	if err != nil {
		t.Fatal(err)
	}
	if source != "local_csv" || mode != "real" || len(returns) != 2 {
		t.Fatalf("unexpected metadata or rows: %s %s %+v", source, mode, returns)
	}
	if math.Abs(returns["2026-01-05"]-0.05) > 1e-10 || math.Abs(returns["2026-01-06"]+0.02) > 1e-10 {
		t.Fatalf("unexpected returns: %+v", returns)
	}
}
