package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpsertPositionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpsertPositionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpsertPositionLogic {
	return &UpsertPositionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpsertPositionLogic) UpsertPosition(in *portfolio.UpsertPositionReq) (*portfolio.PositionResp, error) {
	item, err := l.svcCtx.Service.UpsertPosition(l.ctx, in)
	if err != nil {
		return nil, err
	}
	return &portfolio.PositionResp{Position: item}, nil
}
