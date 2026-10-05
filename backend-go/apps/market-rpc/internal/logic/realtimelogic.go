package logic

import (
	"context"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/repository"
	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type RealtimeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRealtimeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RealtimeLogic {
	return &RealtimeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RealtimeLogic) Realtime(in *market.RealtimeRequest) (*market.RawReply, error) {
	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	if len(in.GetSymbols()) == 0 {
		if rows, err := l.svcCtx.Market.ListQuotes(l.ctx, nil, limit); err == nil && len(rows) > 0 {
			return &market.RawReply{Json: successEnvelope(quotePayload(rows, "database"))}, nil
		}
	}
	if len(in.GetSymbols()) > 0 && l.svcCtx.Redis != nil {
		cachedRows := make([]repository.QuoteRow, 0, len(in.GetSymbols()))
		allCached := true
		for _, symbol := range in.GetSymbols() {
			cached, err := l.svcCtx.Redis.Get(l.ctx, "market:realtime:"+symbol).Result()
			if err != nil || cached == "" || !rawSuccess(cached) {
				allCached = false
				break
			}
			cachedRows = append(cachedRows, parseQuotes(rawData(cached))...)
		}
		if allCached && len(cachedRows) > 0 {
			return &market.RawReply{Json: successEnvelope(quotePayload(cachedRows, "cache"))}, nil
		}
	}
	jsonText, err := l.svcCtx.QuantPython.Realtime(l.ctx, in.GetSymbols(), in.GetLimit())
	if err != nil {
		return l.realtimeListFallback(in.GetSymbols(), limit, "", err)
	}
	if !rawSuccess(jsonText) {
		return l.realtimeListFallback(in.GetSymbols(), limit, jsonText, nil)
	}
	quotes := parseQuotes(rawData(jsonText))
	_ = l.svcCtx.Market.UpsertQuotes(l.ctx, quotes)
	if l.svcCtx.Redis != nil {
		ttl := l.svcCtx.Config.MarketCache.RealtimeTTL
		if ttl <= 0 {
			ttl = 5 * time.Second
		}
		staleTTL := time.Duration(l.svcCtx.Config.MarketCache.StaleCacheMaxSeconds) * time.Second
		if staleTTL <= 0 {
			staleTTL = 300 * time.Second
		}
		for _, quote := range quotes {
			single := successEnvelope(quotePayload([]repository.QuoteRow{quote}, "real-time"))
			_ = l.svcCtx.Redis.Set(l.ctx, "market:realtime:"+quote.Symbol, single, ttl).Err()
			_ = l.svcCtx.Redis.Set(l.ctx, "market:realtime:stale:"+quote.Symbol, single, staleTTL).Err()
		}
	}
	cacheAndPublishQuote(l.ctx, l.svcCtx, "market:realtime:all", jsonText, 0)
	return &market.RawReply{Json: jsonText}, nil
}

func (l *RealtimeLogic) realtimeListFallback(symbols []string, limit int, sourceError string, err error) (*market.RawReply, error) {
	if rows, dbErr := l.svcCtx.Market.ListQuotes(l.ctx, symbols, limit); dbErr == nil && len(rows) > 0 {
		return &market.RawReply{Json: successEnvelope(quotePayload(rows, "database"))}, nil
	}
	if sourceError != "" {
		return &market.RawReply{Json: sourceError}, nil
	}
	message := "source_unavailable"
	if err != nil {
		message = err.Error()
	}
	return &market.RawReply{Json: unavailableEnvelope("source_unavailable: 所有真实行情源不可用", []map[string]string{{"source": "quant-python", "error": message}})}, nil
}
