package logic

import (
	"testing"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/repository"
)

func TestRequestNeedsRealtimeBar(t *testing.T) {
	now := time.Date(2026, time.July, 17, 10, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	if !requestNeedsRealtimeBar("2026-07-17", now) {
		t.Fatal("current trading day must refresh from quant-python")
	}
	if requestNeedsRealtimeBar("2026-07-16", now) {
		t.Fatal("historical range should use database cache")
	}
}

func TestMergeRealtimeQuoteIntoBarsUsesAdjustmentFactor(t *testing.T) {
	now := time.Date(2026, time.July, 17, 10, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	bars := []repository.DailyBarRow{{Symbol: "600176", TradeDate: "2026-07-16", Close: 9, Adjust: "qfq"}}
	quote := &repository.QuoteRow{
		Symbol: "600176", Price: 10.2, Open: 10, High: 10.5, Low: 9.8, PreClose: 10,
		Volume: 2000, Amount: 20400, QuoteTime: now, Source: "tencent", DataMode: "real",
	}
	result := mergeRealtimeQuoteIntoBars(bars, quote, now)
	if len(result) != 2 {
		t.Fatalf("expected current intraday bar, got %d", len(result))
	}
	current := result[1]
	if current.TradeDate != "2026-07-17" || current.Close != 9.18 || !current.IsRealtime {
		t.Fatalf("unexpected intraday bar: %#v", current)
	}
}

func TestMergeRealtimeQuoteIntoBarsRejectsStaleQuote(t *testing.T) {
	now := time.Date(2026, time.July, 17, 10, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	bars := []repository.DailyBarRow{{Symbol: "600176", TradeDate: "2026-07-16", Close: 9}}
	quote := &repository.QuoteRow{Symbol: "600176", Price: 10, PreClose: 9, QuoteTime: now.Add(-24 * time.Hour)}
	if result := mergeRealtimeQuoteIntoBars(bars, quote, now); len(result) != 1 {
		t.Fatalf("stale quote must not create a bar: %#v", result)
	}
}

func TestParseBarsSkipsIntradaySnapshot(t *testing.T) {
	data := map[string]any{"items": []any{
		map[string]any{"symbol": "600176", "trade_date": "2026-07-16", "close": 9.0, "source": "akshare"},
		map[string]any{"symbol": "600176", "trade_date": "2026-07-17", "close": 9.2, "source": "tencent", "is_realtime_bar": true},
	}}
	bars := parseBars(data, "qfq")
	if len(bars) != 1 || bars[0].TradeDate != "2026-07-16" {
		t.Fatalf("only closed bars may be persisted, got %#v", bars)
	}
}

func TestParseTimeTreatsNaiveMarketTimestampAsShanghaiTime(t *testing.T) {
	parsed := parseTime("2026-07-17T22:53:47")
	_, offset := parsed.Zone()
	if offset != 8*60*60 || parsed.Format("2006-01-02 15:04:05") != "2026-07-17 22:53:47" {
		t.Fatalf("unexpected market timestamp: %s offset=%d", parsed, offset)
	}
}
