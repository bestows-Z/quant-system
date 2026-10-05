package logic

import (
	"context"
	"strings"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AddWatchlistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddWatchlistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddWatchlistLogic {
	return &AddWatchlistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AddWatchlistLogic) AddWatchlist(in *market.AddWatchlistRequest) (*market.WatchlistItem, error) {
	if in.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id 不能为空")
	}
	symbol := strings.TrimSpace(in.GetSymbol())
	if symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol 不能为空")
	}
	item, err := l.svcCtx.Watchlists.Add(l.ctx, in.GetUserId(), symbol, normalizeGroup(in.GetGroupName()), strings.TrimSpace(in.GetRemark()))
	if err != nil {
		return nil, err
	}
	return watchlistItem(item), nil
}
