package logic

import "quant-system/backend-go/apps/market-rpc/internal/repository"

func fullIndicatorsFromBars(bars []repository.DailyBarRow) map[string]any {
	closes := make([]float64, len(bars))
	for i, bar := range bars {
		closes[i] = bar.Close
	}
	return map[string]any{
		"ma":   maRows(bars, closes),
		"ema":  emaRows(bars, closes),
		"macd": macdRows(bars, closes),
		"rsi":  rsiRows(bars, closes),
		"boll": bollRows(bars, closes),
		"kdj":  kdjRows(bars),
	}
}

func maRows(bars []repository.DailyBarRow, closes []float64) []map[string]any {
	rows := make([]map[string]any, len(bars))
	for i, bar := range bars {
		rows[i] = map[string]any{"trade_date": bar.TradeDate, "date": bar.TradeDate, "ma5": avg(closes, i, 5), "ma10": avg(closes, i, 10), "ma20": avg(closes, i, 20), "ma60": avg(closes, i, 60)}
	}
	return rows
}

func emaRows(bars []repository.DailyBarRow, closes []float64) []map[string]any {
	return []map[string]any{
		{"period": 12, "items": emaSeries(bars, closes, 12)},
		{"period": 26, "items": emaSeries(bars, closes, 26)},
	}
}

func emaSeries(bars []repository.DailyBarRow, values []float64, period int) []map[string]any {
	rows := make([]map[string]any, len(values))
	if len(values) == 0 {
		return rows
	}
	alpha := 2.0 / float64(period+1)
	ema := values[0]
	for i, value := range values {
		if i > 0 {
			ema = alpha*value + (1-alpha)*ema
		}
		rows[i] = map[string]any{"trade_date": bars[i].TradeDate, "date": bars[i].TradeDate, "ema": round4(ema)}
	}
	return rows
}

func macdRows(bars []repository.DailyBarRow, closes []float64) []map[string]any {
	rows := make([]map[string]any, len(closes))
	if len(closes) == 0 {
		return rows
	}
	ema12, ema26, dea := closes[0], closes[0], 0.0
	for i, close := range closes {
		if i > 0 {
			ema12 = (2.0/13.0)*close + (11.0/13.0)*ema12
			ema26 = (2.0/27.0)*close + (25.0/27.0)*ema26
		}
		dif := ema12 - ema26
		dea = (2.0/10.0)*dif + (8.0/10.0)*dea
		rows[i] = map[string]any{"trade_date": bars[i].TradeDate, "date": bars[i].TradeDate, "dif": round4(dif), "dea": round4(dea), "macd": round4((dif - dea) * 2)}
	}
	return rows
}

func rsiRows(bars []repository.DailyBarRow, closes []float64) []map[string]any {
	rows := make([]map[string]any, len(closes))
	for i, bar := range bars {
		rows[i] = map[string]any{"trade_date": bar.TradeDate, "date": bar.TradeDate, "rsi6": rsi(closes, i, 6), "rsi12": rsi(closes, i, 12), "rsi24": rsi(closes, i, 24)}
	}
	return rows
}

func bollRows(bars []repository.DailyBarRow, closes []float64) []map[string]any {
	rows := make([]map[string]any, len(closes))
	for i, bar := range bars {
		mid := avg(closes, i, 20)
		std := stddev(closes, i, 20, mid)
		rows[i] = map[string]any{"trade_date": bar.TradeDate, "date": bar.TradeDate, "middle": mid, "upper": round4(mid + 2*std), "lower": round4(mid - 2*std)}
	}
	return rows
}

func kdjRows(bars []repository.DailyBarRow) []map[string]any {
	rows := make([]map[string]any, len(bars))
	k, d := 50.0, 50.0
	for i, bar := range bars {
		low, high := bar.Low, bar.High
		start := i - 8
		if start < 0 {
			start = 0
		}
		for j := start; j <= i; j++ {
			if bars[j].Low < low {
				low = bars[j].Low
			}
			if bars[j].High > high {
				high = bars[j].High
			}
		}
		rsv := 50.0
		if high > low {
			rsv = (bar.Close - low) / (high - low) * 100
		}
		k = (2.0/3.0)*k + (1.0/3.0)*rsv
		d = (2.0/3.0)*d + (1.0/3.0)*k
		rows[i] = map[string]any{"trade_date": bar.TradeDate, "date": bar.TradeDate, "k": round4(k), "d": round4(d), "j": round4(3*k - 2*d)}
	}
	return rows
}

func avg(values []float64, idx, period int) float64 {
	start := idx - period + 1
	if start < 0 {
		start = 0
	}
	sum := 0.0
	for i := start; i <= idx; i++ {
		sum += values[i]
	}
	return round4(sum / float64(idx-start+1))
}

func stddev(values []float64, idx, period int, mean float64) float64 {
	start := idx - period + 1
	if start < 0 {
		start = 0
	}
	sum := 0.0
	for i := start; i <= idx; i++ {
		diff := values[i] - mean
		sum += diff * diff
	}
	return sqrt(sum / float64(idx-start+1))
}

func rsi(values []float64, idx, period int) float64 {
	if idx == 0 {
		return 50
	}
	start := idx - period + 1
	if start < 1 {
		start = 1
	}
	gain, loss := 0.0, 0.0
	for i := start; i <= idx; i++ {
		diff := values[i] - values[i-1]
		if diff >= 0 {
			gain += diff
		} else {
			loss -= diff
		}
	}
	if loss == 0 {
		return 100
	}
	return round4(100 - 100/(1+gain/loss))
}

func round4(value float64) float64 {
	if value == 0 {
		return 0
	}
	return float64(int(value*10000+0.5)) / 10000
}

func sqrt(value float64) float64 {
	if value <= 0 {
		return 0
	}
	x := value
	for i := 0; i < 12; i++ {
		x = 0.5 * (x + value/x)
	}
	return x
}
