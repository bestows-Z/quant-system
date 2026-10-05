package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMatchDepthConsumesVisibleLevels(t *testing.T) {
	book := &orderBook{
		Asks: []depthLevel{
			{Level: 1, Price: decimal.RequireFromString("10.01"), Volume: 100},
			{Level: 2, Price: decimal.RequireFromString("10.02"), Volume: 200},
		},
	}
	fills := matchDepth(book, "BUY", "MARKET", decimal.Zero, 250)
	qty, amount, avg := fillSummary(fills)
	if qty != 250 || len(fills) != 2 {
		t.Fatalf("expected two fills totaling 250, got %#v", fills)
	}
	if !amount.Equal(decimal.RequireFromString("2504")) || !avg.Equal(decimal.RequireFromString("10.016")) {
		t.Fatalf("unexpected amount=%s avg=%s", amount, avg)
	}
}

func TestMatchDepthHonorsLimit(t *testing.T) {
	book := &orderBook{
		Asks: []depthLevel{
			{Level: 1, Price: decimal.RequireFromString("10.01"), Volume: 100},
			{Level: 2, Price: decimal.RequireFromString("10.02"), Volume: 200},
		},
	}
	fills := matchDepth(book, "BUY", "LIMIT", decimal.RequireFromString("10.01"), 300)
	qty, _, _ := fillSummary(fills)
	if qty != 100 || len(fills) != 1 {
		t.Fatalf("limit order crossed an ineligible level: %#v", fills)
	}
}

func TestConsumeDepthPreventsVisibleLiquidityReuse(t *testing.T) {
	book := &orderBook{Asks: []depthLevel{{Level: 1, Price: decimal.RequireFromString("10.01"), Volume: 100}}}
	first := matchDepth(book, "BUY", "LIMIT", decimal.RequireFromString("10.01"), 80)
	consumeDepth(book, "BUY", first)
	second := matchDepth(book, "BUY", "LIMIT", decimal.RequireFromString("10.01"), 80)
	qty, _, _ := fillSummary(second)
	if qty != 20 {
		t.Fatalf("expected only 20 shares of visible liquidity remaining, got %d", qty)
	}
}

func TestMatchDepthRejectsZeroVisibleLiquidity(t *testing.T) {
	book := &orderBook{Asks: []depthLevel{{Level: 1, Price: decimal.RequireFromString("10.01"), Volume: 0}}}
	fills := matchDepth(book, "BUY", "MARKET", decimal.Zero, 100)
	qty, _, _ := fillSummary(fills)
	if qty != 0 || len(fills) != 0 {
		t.Fatalf("zero visible liquidity must not produce fills: %#v", fills)
	}
}

func TestNewerQuoteTimeRejectsDuplicateAndOlderSnapshots(t *testing.T) {
	previous := "2026-07-20 10:00:02"
	if newerQuoteTime(previous, previous) {
		t.Fatal("duplicate quote must be rejected")
	}
	if newerQuoteTime(previous, "2026-07-20 10:00:01") {
		t.Fatal("older quote must be rejected")
	}
	if !newerQuoteTime(previous, "2026-07-20 10:00:03") {
		t.Fatal("newer quote must be accepted")
	}
}
