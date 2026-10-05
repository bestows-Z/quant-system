package domain

import (
	"testing"
	"time"
)

func TestTradingDateReached(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	friday := time.Date(2026, 7, 17, 10, 0, 0, 0, location)
	if tradingDateReached(friday, time.Date(2026, 7, 18, 10, 0, 0, 0, location)) {
		t.Fatal("Friday purchase must not settle on Saturday")
	}
	if !tradingDateReached(friday, time.Date(2026, 7, 20, 9, 0, 0, 0, location)) {
		t.Fatal("Friday purchase must settle on Monday")
	}
	monday := time.Date(2026, 7, 20, 10, 0, 0, 0, location)
	if tradingDateReached(monday, monday) {
		t.Fatal("same-day purchase must remain unavailable")
	}
}
