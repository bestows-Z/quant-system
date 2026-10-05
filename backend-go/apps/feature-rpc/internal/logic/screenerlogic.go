package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ScreenerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewScreenerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ScreenerLogic {
	return &ScreenerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ScreenerLogic) Screener(in *feature.ScreenerReq) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.Screener(l.ctx, in.SortBy, in.Order, in.Industry, in.MinChange, in.MaxChange, in.MinAmount, in.Limit)
	return &feature.RawReply{Json: raw}, err
}
