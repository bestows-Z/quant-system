package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarketOverviewLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarketOverviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarketOverviewLogic {
	return &MarketOverviewLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *MarketOverviewLogic) MarketOverview(in *feature.Empty) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.MarketOverview(l.ctx)
	return &feature.RawReply{Json: raw}, err
}
