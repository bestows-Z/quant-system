package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/ws-api/internal/config"
	"quant-system/backend-go/apps/ws-api/internal/hub"
	"quant-system/backend-go/common/markettime"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const cacheTTL = 15 * time.Second

type MarketStreamer struct {
	market marketclient.Market
	paper  paperclient.Paper
	redis  *redis.Client
	hub    *hub.Hub
	conf   config.StreamConf

	mu            sync.Mutex
	sequence      map[string]uint64
	lastPoll      map[string]time.Time
	lastTimeshare map[string]time.Time
	inFlight      map[string]bool
}

func NewMarketStreamer(market marketclient.Market, paper paperclient.Paper, cache *redis.Client, h *hub.Hub, conf config.StreamConf) *MarketStreamer {
	if conf.PollInterval <= 0 {
		conf.PollInterval = time.Second
	}
	if conf.TimeshareInterval <= 0 {
		conf.TimeshareInterval = 5 * time.Second
	}
	if conf.IdleInterval <= 0 {
		conf.IdleInterval = time.Minute
	}
	if conf.MaxConcurrency <= 0 {
		conf.MaxConcurrency = 8
	}
	return &MarketStreamer{
		market: market, paper: paper, redis: cache, hub: h, conf: conf,
		sequence: make(map[string]uint64), lastPoll: make(map[string]time.Time), lastTimeshare: make(map[string]time.Time), inFlight: make(map[string]bool),
	}
}

func (s *MarketStreamer) Start(ctx context.Context) {
	ticker := time.NewTicker(s.conf.PollInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				s.poll(ctx, now)
			}
		}
	}()
}

func (s *MarketStreamer) poll(ctx context.Context, now time.Time) {
	symbols := s.hub.ActiveSymbols("market")
	if len(symbols) == 0 {
		return
	}
	sem := make(chan struct{}, s.conf.MaxConcurrency)
	for _, symbol := range symbols {
		if !s.shouldPoll(symbol, now) {
			continue
		}
		symbol := symbol
		sem <- struct{}{}
		go func() {
			defer func() { s.finishPoll(symbol); <-sem }()
			s.pollSymbol(ctx, symbol, now)
		}()
	}
}

func (s *MarketStreamer) shouldPoll(symbol string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inFlight[symbol] {
		return false
	}
	if !markettime.IsContinuousAuction(now) && now.Sub(s.lastPoll[symbol]) < s.conf.IdleInterval {
		return false
	}
	s.lastPoll[symbol] = now
	s.inFlight[symbol] = true
	return true
}

func (s *MarketStreamer) finishPoll(symbol string) {
	s.mu.Lock()
	delete(s.inFlight, symbol)
	s.mu.Unlock()
}

func (s *MarketStreamer) pollSymbol(parent context.Context, symbol string, now time.Time) {
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		reply, err := s.market.RealtimeBySymbol(ctx, &marketclient.SymbolRequest{Symbol: symbol})
		if err != nil {
			logx.Errorf("ws realtime stream failed symbol=%s err=%v", symbol, err)
			return
		}
		payload, meta, err := decodePayload(reply.GetJson(), true)
		if err != nil {
			logx.Errorf("ws realtime decode failed symbol=%s err=%v", symbol, err)
			return
		}
		s.publish(ctx, "quote", symbol, payload, meta, now)
	}()
	go func() {
		defer wg.Done()
		reply, err := s.market.OrderBook(ctx, &marketclient.SymbolRequest{Symbol: symbol})
		if err != nil {
			logx.Errorf("ws orderbook stream failed symbol=%s err=%v", symbol, err)
			return
		}
		payload, meta, err := decodePayload(reply.GetJson(), false)
		if err != nil {
			logx.Errorf("ws orderbook decode failed symbol=%s err=%v", symbol, err)
			return
		}
		s.publish(ctx, "orderbook", symbol, payload, meta, now)
		if s.paper != nil && markettime.IsContinuousAuction(now) {
			if _, matchErr := s.paper.MatchPendingBySymbol(ctx, &paperclient.MatchPendingBySymbolReq{Symbol: symbol, OrderBookJson: reply.GetJson()}); matchErr != nil {
				logx.Errorf("paper pending match failed symbol=%s err=%v", symbol, matchErr)
			}
		}
	}()
	wg.Wait()

	if !s.timeshareDue(symbol, now) {
		return
	}
	reply, err := s.market.Timeshare(ctx, &marketclient.SymbolRequest{Symbol: symbol})
	if err != nil {
		logx.Errorf("ws timeshare stream failed symbol=%s err=%v", symbol, err)
		return
	}
	payload, meta, err := decodePayload(reply.GetJson(), false)
	if err != nil {
		logx.Errorf("ws timeshare decode failed symbol=%s err=%v", symbol, err)
		return
	}
	if items, ok := payload["items"].([]any); ok && len(items) > 0 {
		if item, valid := items[len(items)-1].(map[string]any); valid {
			meta.QuoteTime = stringValue(item["time"], item["quote_time"])
		}
		payload = map[string]any{"symbol": symbol, "item": items[len(items)-1], "count": len(items)}
	}
	s.publish(ctx, "timeshare", symbol, payload, meta, now)
}

func (s *MarketStreamer) timeshareDue(symbol string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Sub(s.lastTimeshare[symbol]) < s.conf.TimeshareInterval {
		return false
	}
	s.lastTimeshare[symbol] = now
	return true
}

func (s *MarketStreamer) publish(ctx context.Context, feed, symbol string, payload map[string]any, meta payloadMeta, started time.Time) {
	now := time.Now()
	sequence := s.nextSequence(feed + ":" + symbol)
	message := hub.Message{
		Type: "market." + feed + ".updated", Channel: "market", Data: payload,
		CreatedAt: now, QuoteTime: meta.QuoteTime, ServerTime: now.Format("2006-01-02 15:04:05.000"),
		LatencyMS: now.Sub(started).Milliseconds(), Sequence: sequence, Source: meta.Source, DataMode: meta.DataMode,
	}
	encoded, err := json.Marshal(message)
	if err == nil && s.redis != nil {
		if cacheErr := s.redis.Set(ctx, fmt.Sprintf("market:stream:%s:%s", feed, symbol), encoded, cacheTTL).Err(); cacheErr != nil {
			logx.Errorf("ws market cache failed feed=%s symbol=%s err=%v", feed, symbol, cacheErr)
		}
	}
	s.hub.BroadcastSubscribed("market", symbol, 0, message)
}

func (s *MarketStreamer) nextSequence(key string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence[key]++
	return s.sequence[key]
}

type payloadMeta struct {
	Source    string
	DataMode  string
	QuoteTime string
}

func decodePayload(raw string, firstItem bool) (map[string]any, payloadMeta, error) {
	var envelope struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return nil, payloadMeta{}, err
	}
	if envelope.Code != 0 {
		return nil, payloadMeta{}, fmt.Errorf("upstream code=%d message=%s", envelope.Code, envelope.Message)
	}
	if envelope.Data == nil {
		return nil, payloadMeta{}, errors.New("upstream data is empty")
	}
	payload := envelope.Data
	if firstItem {
		items, ok := payload["items"].([]any)
		if !ok || len(items) == 0 {
			return nil, payloadMeta{}, errors.New("upstream quote items are empty")
		}
		item, ok := items[0].(map[string]any)
		if !ok {
			return nil, payloadMeta{}, errors.New("upstream quote item is invalid")
		}
		payload = item
	}
	meta := payloadMeta{
		Source:    stringValue(payload["source"], envelope.Data["source"]),
		DataMode:  stringValue(payload["data_mode"], envelope.Data["data_mode"]),
		QuoteTime: stringValue(payload["quote_time"]),
	}
	if meta.QuoteTime == "" {
		if item, ok := payload["item"].(map[string]any); ok {
			meta.QuoteTime = stringValue(item["time"])
		}
	}
	return payload, meta, nil
}

func stringValue(values ...any) string {
	for _, value := range values {
		if text := strings.TrimSpace(fmt.Sprint(value)); value != nil && text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}
