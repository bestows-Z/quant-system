package markettime

import (
	"strings"
	"testing"
	"time"
)

func localTime(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, Location())
}

func TestParseQuoteTimeFormats(t *testing.T) {
	for _, value := range []string{"20260717101405", "2026-07-17 10:14:05", "2026-07-17T02:14:05Z"} {
		parsed, err := ParseQuoteTime(value)
		if err != nil {
			t.Fatalf("parse %s: %v", value, err)
		}
		if parsed.Hour() != 10 || parsed.Minute() != 14 || parsed.Second() != 5 {
			t.Fatalf("unexpected local time for %s: %s", value, parsed)
		}
	}
}

func TestValidateFreshQuote(t *testing.T) {
	tests := []struct {
		name  string
		now   time.Time
		quote string
		want  string
	}{
		{name: "fresh", now: localTime(2026, time.July, 17, 10, 0, 2), quote: "20260717100000"},
		{name: "stale", now: localTime(2026, time.July, 17, 10, 0, 4), quote: "20260717100000", want: "过期"},
		{name: "future", now: localTime(2026, time.July, 17, 10, 0, 0), quote: "20260717100002", want: "晚于"},
		{name: "lunch", now: localTime(2026, time.July, 17, 12, 0, 0), quote: "20260717120000", want: "连续竞价"},
		{name: "weekend", now: localTime(2026, time.July, 18, 10, 0, 0), quote: "20260718100000", want: "连续竞价"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFreshQuote(tt.now, tt.quote, 3*time.Second)
			if tt.want == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestContinuousAuctionExcludesClosingAuctionAndLunchBoundary(t *testing.T) {
	for _, value := range []string{"2026-07-20 11:30:00", "2026-07-20 14:57:00", "2026-07-20 15:00:00"} {
		at, err := time.ParseInLocation("2006-01-02 15:04:05", value, Location())
		if err != nil {
			t.Fatal(err)
		}
		if IsContinuousAuction(at) {
			t.Fatalf("%s must not be treated as continuous auction", value)
		}
	}
}

func TestOfficial2026HolidayCalendar(t *testing.T) {
	holiday, _ := time.ParseInLocation("2006-01-02", "2026-02-23", Location())
	if IsTradingDay(holiday) {
		t.Fatal("SSE Spring Festival closure must not be a trading day")
	}
	friday, _ := time.ParseInLocation("2006-01-02", "2026-02-13", Location())
	if got := NextTradingDay(friday).Format("2006-01-02"); got != "2026-02-24" {
		t.Fatalf("unexpected next trading day: %s", got)
	}
}
