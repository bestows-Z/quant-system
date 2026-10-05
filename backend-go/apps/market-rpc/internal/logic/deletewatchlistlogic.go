package logic

import (
	"context"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteWatchlistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteWatchlistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteWatchlistLogic {
	return &DeleteWatchlistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteWatchlistLogic) DeleteWatchlist(in *market.DeleteWatchlistRequest) (*market.Empty, error) {
	if in.GetUserId() == 0 || in.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id 和 id 不能为空")
	}
	if err := l.svcCtx.Watchlists.Delete(l.ctx, in.GetUserId(), in.GetId()); err != nil {
		return nil, err
	}
	return &market.Empty{}, nil
}
