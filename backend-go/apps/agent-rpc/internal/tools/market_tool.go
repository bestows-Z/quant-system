package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/market-rpc/marketclient"
)

type MarketTool struct{ Market marketclient.Market }

func (t MarketTool) Name() string { return "market_tool" }
func (t MarketTool) Description() string {
	return "获取股票详情、实时行情、K线、公司资料、公告和财经资讯"
}

func (t MarketTool) Execute(ctx context.Context, input domain.ToolInput) (domain.ToolOutput, error) {
	start := time.Now()
	data := map[string]any{}
	statuses := map[string]string{}
	var mu sync.Mutex
	type operation struct {
		name     string
		call     func() (string, error)
		research bool
	}
	operations := []operation{
		{name: "stock_detail", call: func() (string, error) {
			reply, err := t.Market.StockDetail(ctx, &marketclient.SymbolRequest{Symbol: input.Symbol})
			if err != nil {
				return "", err
			}
			return reply.Json, nil
		}},
		{name: "realtime", call: func() (string, error) {
			reply, err := t.Market.RealtimeBySymbol(ctx, &marketclient.SymbolRequest{Symbol: input.Symbol})
			if err != nil {
				return "", err
			}
			return reply.Json, nil
		}},
		{name: "kline", call: func() (string, error) {
			reply, err := t.Market.Kline(ctx, &marketclient.KlineRequest{Symbol: input.Symbol, Period: "daily"})
			if err != nil {
				return "", err
			}
			return reply.Json, nil
		}},
		{name: "company_profile", call: func() (string, error) {
			reply, err := t.Market.CompanyProfile(ctx, &marketclient.SymbolRequest{Symbol: input.Symbol})
			if err != nil {
				return "", err
			}
			return reply.Json, nil
		}},
		{name: "announcements", research: true, call: func() (string, error) {
			reply, err := t.Market.Announcements(ctx, &marketclient.ResearchRequest{Symbol: input.Symbol, Limit: 20})
			if err != nil {
				return "", err
			}
			return reply.Json, nil
		}},
		{name: "news", research: true, call: func() (string, error) {
			reply, err := t.Market.News(ctx, &marketclient.ResearchRequest{Symbol: input.Symbol, Limit: 20})
			if err != nil {
				return "", err
			}
			return reply.Json, nil
		}},
	}
	var wg sync.WaitGroup
	for _, op := range operations {
		op := op
		wg.Add(1)
		go func() {
			defer wg.Done()
			raw, err := op.call()
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				statuses[op.name] = "SOURCE_UNAVAILABLE: " + err.Error()
				return
			}
			value := rawJSON(raw)
			if op.research {
				value = deduplicateResearch(value, 20)
			}
			data[op.name] = value
			statuses[op.name] = "AVAILABLE"
		}()
	}
	wg.Wait()
	data["source_status"] = statuses
	success := len(data) > 1
	return domain.ToolOutput{ToolName: t.Name(), Data: data, Success: success, LatencyMs: time.Since(start).Milliseconds()}, nil
}

func rawJSON(value string) any {
	var out any
	if json.Unmarshal([]byte(value), &out) == nil {
		return out
	}
	return value
}

func deduplicateResearch(value any, limit int) any {
	root, ok := value.(map[string]any)
	if !ok {
		return value
	}
	data, ok := root["data"].(map[string]any)
	if !ok {
		return value
	}
	items, ok := data["items"].([]any)
	if !ok {
		return value
	}
	seen := map[string]bool{}
	result := make([]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(fmt.Sprint(item["url"])))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(fmt.Sprint(item["title"]))) + "|" + strings.TrimSpace(fmt.Sprint(item["published_at"]))
		}
		if key == "|" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}
	data["items"], data["count"] = result, len(result)
	root["data"] = data
	return root
}
