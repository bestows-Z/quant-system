package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPortfolioLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPortfolioLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPortfolioLogic {
	return &GetPortfolioLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPortfolioLogic) GetPortfolio(in *portfolio.GetPortfolioReq) (*portfolio.PortfolioResp, error) {
	item, err := l.svcCtx.Service.GetPortfolio(l.ctx, in.GetUserId(), in.GetPortfolioId())
	if err != nil {
		return nil, err
	}
	return &portfolio.PortfolioResp{Portfolio: item}, nil
}
