package logic

import (
	"context"
	"math"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/repository"
	"quant-system/backend-go/apps/market-rpc/internal/svc"
)

func loadBarsWithRealtimeSnapshot(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	symbol, startDate, endDate, adjust string,
) []repository.DailyBarRow {
	bars, err := svcCtx.Market.ListBars(ctx, symbol, startDate, endDate, adjust)
	if err != nil || len(bars) == 0 {
		return nil
	}
	if !requestNeedsRealtimeBar(endDate, time.Now()) {
		return bars
	}
	return mergeRealtimeQuoteIntoBars(bars, latestCurrentQuote(ctx, svcCtx, symbol), time.Now())
}

func latestCurrentQuote(ctx context.Context, svcCtx *svc.ServiceContext, symbol string) *repository.QuoteRow {
	if jsonText, err := svcCtx.QuantPython.RealtimeBySymbol(ctx, symbol); err == nil && rawSuccess(jsonText) {
		quotes := parseQuotes(rawData(jsonText))
		for index := range quotes {
			if quotes[index].Symbol == symbol && quoteMatchesShanghaiDate(quotes[index].QuoteTime, time.Now()) {
				_ = svcCtx.Market.UpsertQuotes(ctx, quotes)
				return &quotes[index]
			}
		}
	}
	quote, err := svcCtx.Market.GetQuote(ctx, symbol)
	if err != nil || !quoteMatchesShanghaiDate(quote.QuoteTime, time.Now()) {
		return nil
	}
	return quote
}

func mergeRealtimeQuoteIntoBars(bars []repository.DailyBarRow, quote *repository.QuoteRow, now time.Time) []repository.DailyBarRow {
	if len(bars) == 0 || quote == nil || quote.Price <= 0 || quote.PreClose <= 0 || !quoteMatchesShanghaiDate(quote.QuoteTime, now) {
		return bars
	}
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	today := now.In(shanghai).Format("2006-01-02")
	closed := make([]repository.DailyBarRow, 0, len(bars)+1)
	for _, bar := range bars {
		if bar.TradeDate != today {
			closed = append(closed, bar)
		}
	}
	if len(closed) == 0 {
		return bars
	}
	previous := closed[len(closed)-1]
	factor := previous.Close / quote.PreClose
	current := repository.DailyBarRow{
		Symbol:        quote.Symbol,
		TradeDate:     today,
		Open:          roundPrice(valueOrFloat(quote.Open, quote.Price) * factor),
		High:          roundPrice(valueOrFloat(quote.High, quote.Price) * factor),
		Low:           roundPrice(valueOrFloat(quote.Low, quote.Price) * factor),
		Close:         roundPrice(quote.Price * factor),
		PreClose:      roundPrice(previous.Close),
		Volume:        quote.Volume,
		Amount:        quote.Amount,
		TurnoverRate:  quote.TurnoverRate,
		ChangePercent: quote.Price/quote.PreClose - 1,
		ChangeAmount:  roundPrice((quote.Price - quote.PreClose) * factor),
		Amplitude:     (valueOrFloat(quote.High, quote.Price) - valueOrFloat(quote.Low, quote.Price)) / quote.PreClose,
		Adjust:        valueOr(previous.Adjust, "qfq"),
		Source:        quote.Source,
		DataMode:      valueOr(quote.DataMode, "real"),
		IsRealtime:    true,
		BarStatus:     "INTRADAY",
		QuoteTime:     quote.QuoteTime,
	}
	return append(closed, current)
}

func quoteMatchesShanghaiDate(quoteTime, now time.Time) bool {
	if quoteTime.IsZero() {
		return false
	}
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return quoteTime.Format("2006-01-02") == now.Format("2006-01-02")
	}
	return quoteTime.In(shanghai).Format("2006-01-02") == now.In(shanghai).Format("2006-01-02")
}

func valueOrFloat(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func roundPrice(value float64) float64 {
	return math.Round(value*10000) / 10000
}
