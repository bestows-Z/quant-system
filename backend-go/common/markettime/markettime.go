package markettime

import (
	"fmt"
	"strings"
	"time"
)

const DefaultMaxQuoteAge = 3 * time.Second

var shanghai = mustShanghaiLocation()

// Exchange holidays are sourced from the SSE annual closure notice. Extend
// this fail-closed table when the exchange publishes the next year's calendar.
var exchangeHolidays = map[string]struct{}{
	"2026-01-01": {}, "2026-01-02": {},
	"2026-02-16": {}, "2026-02-17": {}, "2026-02-18": {}, "2026-02-19": {}, "2026-02-20": {}, "2026-02-23": {},
	"2026-04-06": {},
	"2026-05-01": {}, "2026-05-04": {}, "2026-05-05": {},
	"2026-06-19": {},
	"2026-09-25": {},
	"2026-10-01": {}, "2026-10-02": {}, "2026-10-05": {}, "2026-10-06": {}, "2026-10-07": {},
}

func Location() *time.Location { return shanghai }

func Now() time.Time { return time.Now().In(shanghai) }

func ParseQuoteTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("行情时间为空")
	}
	for _, layout := range []string{"20060102150405", "2006-01-02 15:04:05", "2006-01-02 15:04", time.RFC3339Nano, time.RFC3339} {
		var parsed time.Time
		var err error
		if layout == time.RFC3339Nano || layout == time.RFC3339 {
			parsed, err = time.Parse(layout, value)
		} else {
			parsed, err = time.ParseInLocation(layout, value, shanghai)
		}
		if err == nil {
			return parsed.In(shanghai), nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析行情时间 %q", value)
}

func IsContinuousAuction(now time.Time) bool {
	local := now.In(shanghai)
	if !IsTradingDay(local) {
		return false
	}
	second := local.Hour()*60*60 + local.Minute()*60 + local.Second()
	return (second >= (9*60+30)*60 && second < (11*60+30)*60) || (second >= 13*60*60 && second < (14*60+57)*60)
}

func IsTradingDay(day time.Time) bool {
	local := day.In(shanghai)
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return false
	}
	_, closed := exchangeHolidays[local.Format("2006-01-02")]
	return !closed
}

func NextTradingDay(day time.Time) time.Time {
	local := day.In(shanghai)
	next := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, shanghai).AddDate(0, 0, 1)
	for !IsTradingDay(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func ValidateFreshQuote(now time.Time, quoteTime string, maxAge time.Duration) error {
	if !IsContinuousAuction(now) {
		return fmt.Errorf("当前不在 A 股连续竞价时段")
	}
	quoteAt, err := ParseQuoteTime(quoteTime)
	if err != nil {
		return err
	}
	age := now.In(shanghai).Sub(quoteAt)
	if age < -time.Second {
		return fmt.Errorf("行情时间晚于服务器时间，已拒绝倒序快照")
	}
	if maxAge <= 0 {
		maxAge = DefaultMaxQuoteAge
	}
	if age > maxAge {
		return fmt.Errorf("盘口已过期（%.1f 秒），请等待实时行情更新", age.Seconds())
	}
	return nil
}

func mustShanghaiLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return location
	}
	return time.FixedZone("CST", 8*60*60)
}
