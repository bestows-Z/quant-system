package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPortfoliosLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPortfoliosLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPortfoliosLogic {
	return &ListPortfoliosLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListPortfoliosLogic) ListPortfolios(in *portfolio.ListPortfoliosReq) (*portfolio.ListPortfoliosResp, error) {
	items, err := l.svcCtx.Service.ListPortfolios(l.ctx, in.GetUserId())
	if err != nil {
		return nil, err
	}
	return &portfolio.ListPortfoliosResp{Items: items}, nil
}
