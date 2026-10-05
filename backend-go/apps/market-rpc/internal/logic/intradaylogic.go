package logic

import (
	"context"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type IntradayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIntradayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IntradayLogic {
	return &IntradayLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *IntradayLogic) Timeshare(in *market.SymbolRequest) (*market.RawReply, error) {
	return l.cached("market:timeshare:"+in.GetSymbol(), 3*time.Second, func() (string, error) {
		return l.svcCtx.QuantPython.Timeshare(l.ctx, in.GetSymbol())
	})
}

func (l *IntradayLogic) OrderBook(in *market.SymbolRequest) (*market.RawReply, error) {
	return l.cached("market:orderbook:"+in.GetSymbol(), 2*time.Second, func() (string, error) {
		return l.svcCtx.QuantPython.OrderBook(l.ctx, in.GetSymbol())
	})
}

func (l *IntradayLogic) cached(key string, ttl time.Duration, load func() (string, error)) (*market.RawReply, error) {
	if l.svcCtx.Redis != nil {
		if cached, err := l.svcCtx.Redis.Get(l.ctx, key).Result(); err == nil && cached != "" {
			return &market.RawReply{Json: cached}, nil
		}
	}
	raw, err := load()
	if err != nil {
		return &market.RawReply{Json: unavailableEnvelope("source_unavailable: 实时分时行情源不可用", []map[string]string{{"source": "quant-python", "error": err.Error()}})}, nil
	}
	if rawSuccess(raw) && l.svcCtx.Redis != nil {
		if err := l.svcCtx.Redis.Set(l.ctx, key, raw, ttl).Err(); err != nil {
			l.Logger.Errorf("cache intraday market data failed: %v", err)
		}
	}
	return &market.RawReply{Json: raw}, nil
}
