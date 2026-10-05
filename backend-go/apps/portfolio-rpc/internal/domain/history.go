package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"
)

type klineEnvelope struct {
	Data struct {
		Source   string `json:"source"`
		DataMode string `json:"data_mode"`
		Items    []struct {
			TradeDate string          `json:"trade_date"`
			Date      string          `json:"date"`
			Close     json.RawMessage `json:"close"`
		} `json:"items"`
	} `json:"data"`
}

func (s *Service) historicalAnalysis(ctx context.Context, positions []*portfolio.PositionAnalysis, marketValue float64) advancedAnalysis {
	series := make([]historicalSeries, len(positions))
	type meta struct{ source, mode string }
	metadata := make([]meta, len(positions))
	semaphore := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, position := range positions {
		i, position := i, position
		weight := parseFloat(position.Weight)
		series[i] = historicalSeries{Symbol: position.Symbol, Name: position.Name, Industry: position.Industry, Weight: weight, Returns: map[string]float64{}}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}
			requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			end := time.Now().Format("2006-01-02")
			start := time.Now().AddDate(-1, -3, 0).Format("2006-01-02")
			reply, err := s.market.Kline(requestCtx, &marketclient.KlineRequest{Symbol: position.Symbol, Period: "daily", StartDate: start, EndDate: end, Adjust: "qfq"})
			if err != nil {
				return
			}
			returns, source, mode, err := parseKlineReturns(reply.Json)
			if err != nil {
				return
			}
			series[i].Returns = returns
			metadata[i] = meta{source: source, mode: mode}
		}()
	}
	wg.Wait()
	sources, modes := map[string]bool{}, map[string]bool{}
	for _, item := range metadata {
		if item.source != "" {
			sources[item.source] = true
		}
		if item.mode != "" {
			modes[item.mode] = true
		}
	}
	return calculateAdvanced(series, marketValue, sortedKeys(sources), sortedKeys(modes))
}

func parseKlineReturns(raw string) (map[string]float64, string, string, error) {
	var body klineEnvelope
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		return nil, "", "", err
	}
	type point struct {
		date  string
		close float64
	}
	points := make([]point, 0, len(body.Data.Items))
	for _, item := range body.Data.Items {
		date := strings.TrimSpace(item.TradeDate)
		if date == "" {
			date = strings.TrimSpace(item.Date)
		}
		closeValue, ok := rawNumber(item.Close)
		if date != "" && ok && closeValue > 0 {
			points = append(points, point{date: date, close: closeValue})
		}
	}
	if len(points) < 2 {
		return nil, body.Data.Source, body.Data.DataMode, fmt.Errorf("insufficient kline history")
	}
	sort.SliceStable(points, func(i, j int) bool { return points[i].date < points[j].date })
	returns := make(map[string]float64, len(points)-1)
	for i := 1; i < len(points); i++ {
		if points[i-1].close > 0 {
			returns[points[i].date] = points[i].close/points[i-1].close - 1
		}
	}
	return returns, body.Data.Source, body.Data.DataMode, nil
}

func rawNumber(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false
	}
	parsed := parseFloat(fmt.Sprint(value))
	return parsed, parsed != 0
}

func parseFloat(value string) float64 {
	var parsed float64
	_, _ = fmt.Sscan(strings.TrimSpace(value), &parsed)
	return parsed
}

func sortedKeys(values map[string]bool) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}
