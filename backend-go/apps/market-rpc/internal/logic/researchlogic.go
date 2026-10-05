package logic

import (
	"context"
	"fmt"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResearchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResearchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResearchLogic {
	return &ResearchLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ResearchLogic) MarketIndices(_ *market.Empty) (*market.RawReply, error) {
	return l.fetch("research:indices", 15*time.Second, l.svcCtx.QuantPython.MarketIndices)
}

func (l *ResearchLogic) MarketBreadth(_ *market.Empty) (*market.RawReply, error) {
	return l.fetch("research:market-breadth", 15*time.Second, l.svcCtx.QuantPython.MarketBreadth)
}

func (l *ResearchLogic) CompanyProfile(in *market.SymbolRequest) (*market.RawReply, error) {
	return l.fetch("research:company:"+in.GetSymbol(), 24*time.Hour, func(ctx context.Context) (string, error) {
		return l.svcCtx.QuantPython.CompanyProfile(ctx, in.GetSymbol())
	})
}

func (l *ResearchLogic) Announcements(in *market.ResearchRequest) (*market.RawReply, error) {
	key := fmt.Sprintf("research:announcements:%s:%d", in.GetSymbol(), normalizedResearchLimit(in.GetLimit(), 20))
	return l.fetch(key, 5*time.Minute, func(ctx context.Context) (string, error) {
		return l.svcCtx.QuantPython.Announcements(ctx, in.GetSymbol(), normalizedResearchLimit(in.GetLimit(), 20))
	})
}

func (l *ResearchLogic) News(in *market.ResearchRequest) (*market.RawReply, error) {
	key := fmt.Sprintf("research:news:%s:%s:%d", in.GetSymbol(), in.GetKeyword(), normalizedResearchLimit(in.GetLimit(), 20))
	return l.fetch(key, 2*time.Minute, func(ctx context.Context) (string, error) {
		return l.svcCtx.QuantPython.News(ctx, in.GetSymbol(), in.GetKeyword(), normalizedResearchLimit(in.GetLimit(), 20))
	})
}

func (l *ResearchLogic) MoneyFlow(in *market.ResearchRequest) (*market.RawReply, error) {
	key := fmt.Sprintf("research:money-flow:%s:%d", in.GetSymbol(), normalizedResearchLimit(in.GetLimit(), 20))
	return l.fetch(key, 30*time.Second, func(ctx context.Context) (string, error) {
		return l.svcCtx.QuantPython.MoneyFlow(ctx, in.GetSymbol(), normalizedResearchLimit(in.GetLimit(), 20))
	})
}

func (l *ResearchLogic) DragonTiger(in *market.ResearchRequest) (*market.RawReply, error) {
	key := fmt.Sprintf("research:dragon-tiger:%s:%d", in.GetTradeDate(), normalizedResearchLimit(in.GetLimit(), 50))
	return l.fetch(key, 5*time.Minute, func(ctx context.Context) (string, error) {
		return l.svcCtx.QuantPython.DragonTiger(ctx, in.GetTradeDate(), normalizedResearchLimit(in.GetLimit(), 50))
	})
}

func (l *ResearchLogic) DragonTigerSeats(in *market.ResearchRequest) (*market.RawReply, error) {
	key := fmt.Sprintf("research:dragon-tiger:seats:%s:%s", in.GetTradeDate(), in.GetSymbol())
	return l.fetch(key, 30*time.Minute, func(ctx context.Context) (string, error) {
		return l.svcCtx.QuantPython.DragonTigerSeats(ctx, in.GetSymbol(), in.GetTradeDate())
	})
}

func (l *ResearchLogic) fetch(key string, ttl time.Duration, call func(context.Context) (string, error)) (*market.RawReply, error) {
	if l.svcCtx.Redis != nil {
		if cached, err := l.svcCtx.Redis.Get(l.ctx, key).Result(); err == nil && cached != "" {
			return &market.RawReply{Json: cached}, nil
		}
	}
	raw, err := call(l.ctx)
	if err == nil && rawSuccess(raw) {
		if l.svcCtx.Redis != nil {
			_ = l.svcCtx.Redis.Set(l.ctx, key, raw, ttl).Err()
			_ = l.svcCtx.Redis.Set(l.ctx, key+":stale", raw, 24*time.Hour).Err()
		}
		return &market.RawReply{Json: raw}, nil
	}
	if l.svcCtx.Redis != nil {
		if stale, cacheErr := l.svcCtx.Redis.Get(l.ctx, key+":stale").Result(); cacheErr == nil && stale != "" {
			l.Logger.Errorf("research source failed, serving stale cache key=%s err=%v", key, err)
			return &market.RawReply{Json: stale}, nil
		}
	}
	if raw != "" {
		return &market.RawReply{Json: raw}, nil
	}
	return nil, err
}

func normalizedResearchLimit(value, fallback int32) int32 {
	if value <= 0 {
		return fallback
	}
	if value > 100 {
		return 100
	}
	return value
}
