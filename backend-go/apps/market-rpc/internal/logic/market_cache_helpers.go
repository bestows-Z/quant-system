package logic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/repository"
)

func successEnvelope(data any) string {
	raw, _ := json.Marshal(map[string]any{"code": 0, "message": "success", "data": data})
	return string(raw)
}

func unavailableEnvelope(message string, errors []map[string]string) string {
	if message == "" {
		message = "source_unavailable: 所有真实行情源不可用"
	}
	raw, _ := json.Marshal(map[string]any{
		"code":    503,
		"message": message,
		"data": map[string]any{
			"data_mode": "real",
			"source":    "unavailable",
			"errors":    errors,
		},
	})
	return string(raw)
}

func rawSuccess(rawJSON string) bool {
	var body map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &body); err != nil {
		return false
	}
	code, _ := asInt(body["code"])
	return code == 0
}

func rawData(rawJSON string) map[string]any {
	var body map[string]any
	_ = json.Unmarshal([]byte(rawJSON), &body)
	if data, ok := body["data"].(map[string]any); ok {
		return data
	}
	return map[string]any{}
}

func rawErrors(rawJSON string, fallbackSource string) []map[string]string {
	data := rawData(rawJSON)
	items, _ := data["errors"].([]any)
	errors := make([]map[string]string, 0, len(items)+1)
	for _, item := range items {
		if row, ok := item.(map[string]any); ok {
			errors = append(errors, map[string]string{
				"source": fmt.Sprint(row["source"]),
				"error":  fmt.Sprint(row["error"]),
			})
		}
	}
	if len(errors) == 0 {
		errors = append(errors, map[string]string{"source": fallbackSource, "error": "source_unavailable"})
	}
	return errors
}

func withDataSource(data map[string]any, source string) map[string]any {
	data["source"] = source
	data["data_mode"] = modeForSource(source)
	if items, ok := data["items"].([]map[string]any); ok {
		for _, item := range items {
			item["source"] = source
			item["data_mode"] = modeForSource(source)
		}
	}
	return data
}

func stockPayload(items []repository.StockRow, total, page, pageSize int, source string) map[string]any {
	dataMode := modeForSource(source)
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"symbol":       item.Symbol,
			"name":         item.Name,
			"market":       item.Market,
			"exchange":     item.Exchange,
			"industry":     item.Industry,
			"listing_date": nullableDate(item.ListingDate.String),
			"source":       source,
			"data_mode":    dataMode,
			"updated_at":   item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return map[string]any{"items": rows, "count": len(rows), "total": total, "page": page, "page_size": pageSize, "source": source, "data_mode": dataMode}
}

func quotePayload(items []repository.QuoteRow, source string) map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, quoteMap(item, source))
	}
	return map[string]any{"items": rows, "count": len(rows), "source": source, "data_mode": "real"}
}

func quoteMap(item repository.QuoteRow, source string) map[string]any {
	return map[string]any{
		"symbol":         item.Symbol,
		"name":           item.Name,
		"price":          item.Price,
		"current":        item.Price,
		"open":           item.Open,
		"high":           item.High,
		"low":            item.Low,
		"pre_close":      item.PreClose,
		"volume":         item.Volume,
		"amount":         item.Amount,
		"change":         item.ChangeAmount,
		"change_pct":     item.ChangePercent,
		"change_percent": item.ChangePercent,
		"pct_change":     item.ChangePercent,
		"turnover_rate":  item.TurnoverRate,
		"amplitude":      item.Amplitude,
		"volume_ratio":   item.VolumeRatio,
		"pe_ratio":       item.PERatio,
		"limit_up":       item.LimitUp,
		"limit_down":     item.LimitDown,
		"quote_time":     item.QuoteTime.Format(time.RFC3339),
		"source":         source,
		"data_mode":      "real",
		"updated_at":     item.UpdatedAt.Format(time.RFC3339),
	}
}

func klinePayload(symbol, period string, bars []repository.DailyBarRow, source string) map[string]any {
	dataMode := modeForSource(source)
	items := make([]map[string]any, 0, len(bars))
	for _, bar := range bars {
		itemSource := valueOr(bar.Source, source)
		items = append(items, map[string]any{
			"symbol":          bar.Symbol,
			"trade_date":      bar.TradeDate,
			"date":            bar.TradeDate,
			"open":            bar.Open,
			"high":            bar.High,
			"low":             bar.Low,
			"close":           bar.Close,
			"pre_close":       bar.PreClose,
			"volume":          bar.Volume,
			"amount":          bar.Amount,
			"turnover_rate":   bar.TurnoverRate,
			"change_pct":      bar.ChangePercent,
			"change_percent":  bar.ChangePercent,
			"change_amount":   bar.ChangeAmount,
			"amplitude":       bar.Amplitude,
			"adjust":          bar.Adjust,
			"source":          itemSource,
			"data_mode":       dataMode,
			"is_realtime_bar": bar.IsRealtime,
			"bar_status":      valueOr(bar.BarStatus, "CLOSED"),
			"quote_time":      formatOptionalTime(bar.QuoteTime),
		})
	}
	return map[string]any{"symbol": symbol, "period": valueOr(period, "daily"), "items": items, "count": len(items), "source": source, "data_mode": dataMode}
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func modeForSource(source string) string {
	if source == "local_csv" {
		return "bootstrap"
	}
	return "real"
}

func stockStorageSource(items []repository.StockRow) string {
	if len(items) > 0 && items[0].Source == "local_csv" {
		return "local_csv"
	}
	return "database"
}

func barStorageSource(items []repository.DailyBarRow) string {
	if len(items) > 0 && items[0].Source == "local_csv" {
		return "local_csv"
	}
	return "database"
}

func parseStocks(data map[string]any) []repository.StockRow {
	items := mapItems(data["items"])
	rows := make([]repository.StockRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, repository.StockRow{
			Symbol:   asString(item["symbol"]),
			Name:     asString(item["name"]),
			Market:   valueOr(asString(item["market"]), "A股"),
			Exchange: asString(item["exchange"]),
			Industry: asString(item["industry"]),
			Source:   asString(item["source"]),
		})
	}
	return rows
}

func parseQuotes(data map[string]any) []repository.QuoteRow {
	items := mapItems(data["items"])
	rows := make([]repository.QuoteRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, repository.QuoteRow{
			Symbol:        asString(item["symbol"]),
			Name:          asString(item["name"]),
			Price:         asFloat(firstAny(item["price"], item["current"])),
			Open:          asFloat(item["open"]),
			High:          asFloat(item["high"]),
			Low:           asFloat(item["low"]),
			PreClose:      asFloat(item["pre_close"]),
			Volume:        int64(asFloat(item["volume"])),
			Amount:        asFloat(item["amount"]),
			ChangeAmount:  asFloat(firstAny(item["change"], item["change_amount"])),
			ChangePercent: asFloat(firstAny(item["change_pct"], item["pct_change"], item["change_percent"])),
			TurnoverRate:  asFloat(item["turnover_rate"]),
			Amplitude:     asFloat(item["amplitude"]),
			VolumeRatio:   asFloat(item["volume_ratio"]),
			PERatio:       asFloat(item["pe_ratio"]),
			LimitUp:       asFloat(item["limit_up"]),
			LimitDown:     asFloat(item["limit_down"]),
			QuoteTime:     parseTime(asString(item["quote_time"])),
			Source:        asString(item["source"]),
			DataMode:      valueOr(asString(item["data_mode"]), modeForSource(asString(item["source"]))),
		})
	}
	return rows
}

func parseBars(data map[string]any, adjust string) []repository.DailyBarRow {
	items := mapItems(data["items"])
	rows := make([]repository.DailyBarRow, 0, len(items))
	for _, item := range items {
		if isRealtime, _ := item["is_realtime_bar"].(bool); isRealtime {
			continue
		}
		rows = append(rows, repository.DailyBarRow{
			Symbol:        asString(item["symbol"]),
			TradeDate:     firstString(item["trade_date"], item["date"]),
			Open:          asFloat(item["open"]),
			High:          asFloat(item["high"]),
			Low:           asFloat(item["low"]),
			Close:         asFloat(item["close"]),
			PreClose:      asFloat(item["pre_close"]),
			Volume:        int64(asFloat(item["volume"])),
			Amount:        asFloat(item["amount"]),
			TurnoverRate:  asFloat(item["turnover_rate"]),
			ChangePercent: asFloat(firstAny(item["change_pct"], item["change_percent"])),
			ChangeAmount:  asFloat(item["change_amount"]),
			Amplitude:     asFloat(item["amplitude"]),
			Adjust:        valueOr(adjust, "qfq"),
			Source:        asString(item["source"]),
			DataMode:      valueOr(asString(item["data_mode"]), modeForSource(asString(item["source"]))),
		})
	}
	return rows
}

func requestNeedsRealtimeBar(endDate string, now time.Time) bool {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		now = now.In(shanghai)
	}
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	today := now.Format("2006-01-02")
	return endDate == "" || endDate >= today
}

func mapItems(value any) []map[string]any {
	rawItems, _ := value.([]any)
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		if item, ok := raw.(map[string]any); ok {
			items = append(items, item)
		}
	}
	return items
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func firstString(values ...any) string {
	for _, value := range values {
		if text := asString(value); text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func asFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

func asInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

func firstAny(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Now()
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, value, location); err == nil {
			return t
		}
	}
	return time.Now()
}

func nullableDate(value string) string {
	if value == "<nil>" {
		return ""
	}
	return value
}

func valueOr(value, fallback string) string {
	if value == "" || value == "<nil>" {
		return fallback
	}
	return value
}
