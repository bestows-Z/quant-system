package logic

import (
	"context"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type WatchlistsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWatchlistsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WatchlistsLogic {
	return &WatchlistsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *WatchlistsLogic) Watchlists(in *market.WatchlistRequest) (*market.WatchlistReply, error) {
	items, err := l.svcCtx.Watchlists.List(l.ctx, in.GetUserId())
	if err != nil {
		return nil, err
	}
	reply := &market.WatchlistReply{Items: make([]*market.WatchlistItem, 0, len(items))}
	for i := range items {
		reply.Items = append(reply.Items, watchlistItem(&items[i]))
	}
	return reply, nil
}
