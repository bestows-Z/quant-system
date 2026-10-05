package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeletePositionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeletePositionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePositionLogic {
	return &DeletePositionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeletePositionLogic) DeletePosition(in *portfolio.DeletePositionReq) (*portfolio.DeletePositionResp, error) {
	if err := l.svcCtx.Service.DeletePosition(l.ctx, in.GetUserId(), in.GetPortfolioId(), in.GetPositionId()); err != nil {
		return nil, err
	}
	return &portfolio.DeletePositionResp{Deleted: true}, nil
}
