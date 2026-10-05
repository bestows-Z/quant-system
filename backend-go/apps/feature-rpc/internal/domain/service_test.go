package domain

import (
	"testing"
	"time"

	"quant-system/backend-go/common/markettime"
)

func TestCalculateOverviewAndSectorStrength(t *testing.T) {
	quoteTime := time.Date(2026, time.July, 17, 15, 0, 0, 0, markettime.Location())
	rows := []quoteRow{{Industry: "银行", Change: .02, Amount: 100, Price: 10, DataMode: "real", QuoteTime: quoteTime}, {Industry: "银行", Change: -.01, Amount: 50, Price: 9, DataMode: "real", QuoteTime: quoteTime}, {Industry: "材料", Change: .05, Amount: 200, Price: 20, DataMode: "real", QuoteTime: quoteTime}}
	o := calculateOverview(rows)
	if o.RiseCount != 2 || o.FallCount != 1 || o.TotalAmount != 350 {
		t.Fatalf("unexpected overview: %+v", o)
	}
	sectors := calculateSectors(rows)
	if len(sectors) != 2 || sectors[0].Name != "材料" {
		t.Fatalf("unexpected sectors: %+v", sectors)
	}
}
func TestSortQuotesUsesAllowlist(t *testing.T) {
	rows := []quoteRow{{Symbol: "A", Change: .01, Amount: 200}, {Symbol: "B", Change: .02, Amount: 100}}
	sortQuotes(rows, "amount", "asc")
	if rows[0].Symbol != "B" {
		t.Fatalf("unexpected sort: %+v", rows)
	}
	if normalizeSort("price desc;drop table") != "change_percent" {
		t.Fatal("sort allowlist bypass")
	}
}
func TestNormalizePercent(t *testing.T) {
	if normalizePct(9.8) != .098 || normalizePct(.05) != .05 {
		t.Fatal("percent normalization failed")
	}
}
