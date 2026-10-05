package logic

import (
	"context"
	"database/sql"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/repository"
	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type RealtimeBySymbolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRealtimeBySymbolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RealtimeBySymbolLogic {
	return &RealtimeBySymbolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RealtimeBySymbolLogic) RealtimeBySymbol(in *market.SymbolRequest) (*market.RawReply, error) {
	symbol := in.GetSymbol()
	cacheKey := "market:realtime:" + symbol
	staleKey := "market:realtime:stale:" + symbol
	if l.svcCtx.Redis != nil {
		if cached, err := l.svcCtx.Redis.Get(l.ctx, cacheKey).Result(); err == nil && cached != "" {
			return &market.RawReply{Json: successEnvelope(withDataSource(rawData(cached), "cache"))}, nil
		}
	}
	jsonText, err := l.svcCtx.QuantPython.RealtimeBySymbol(l.ctx, in.GetSymbol())
	if err != nil {
		return l.realtimeFallback(symbol, staleKey, "", err)
	}
	if !rawSuccess(jsonText) {
		return l.realtimeFallback(symbol, staleKey, jsonText, nil)
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
		_ = l.svcCtx.Redis.Set(l.ctx, cacheKey, jsonText, ttl).Err()
		_ = l.svcCtx.Redis.Set(l.ctx, staleKey, jsonText, staleTTL).Err()
	}
	cacheAndPublishQuote(l.ctx, l.svcCtx, cacheKey, jsonText, 0)
	return &market.RawReply{Json: jsonText}, nil
}

func (l *RealtimeBySymbolLogic) realtimeFallback(symbol, staleKey, sourceError string, err error) (*market.RawReply, error) {
	if l.svcCtx.Config.MarketCache.AllowStaleCache && l.svcCtx.Redis != nil {
		if cached, getErr := l.svcCtx.Redis.Get(l.ctx, staleKey).Result(); getErr == nil && cached != "" {
			return &market.RawReply{Json: successEnvelope(withDataSource(rawData(cached), "stale-cache"))}, nil
		} else if getErr != nil && getErr != redis.Nil {
			l.Logger.Errorf("read stale quote cache failed: %v", getErr)
		}
	}
	if row, dbErr := l.svcCtx.Market.GetQuote(l.ctx, symbol); dbErr == nil {
		return &market.RawReply{Json: successEnvelope(quotePayload([]repository.QuoteRow{*row}, "database"))}, nil
	} else if dbErr != sql.ErrNoRows {
		l.Logger.Errorf("read realtime quote db failed: %v", dbErr)
	}
	if sourceError != "" {
		return &market.RawReply{Json: sourceError}, nil
	}
	return &market.RawReply{Json: unavailableEnvelope("source_unavailable: 所有真实行情源不可用", []map[string]string{{"source": "quant-python", "error": err.Error()}})}, nil
}
