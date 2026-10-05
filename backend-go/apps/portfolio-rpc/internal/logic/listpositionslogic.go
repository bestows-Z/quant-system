package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPositionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPositionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPositionsLogic {
	return &ListPositionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListPositionsLogic) ListPositions(in *portfolio.ListPositionsReq) (*portfolio.ListPositionsResp, error) {
	items, err := l.svcCtx.Service.ListPositions(l.ctx, in.GetUserId(), in.GetPortfolioId())
	if err != nil {
		return nil, err
	}
	return &portfolio.ListPositionsResp{Items: items}, nil
}
