package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePortfolioLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreatePortfolioLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePortfolioLogic {
	return &CreatePortfolioLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreatePortfolioLogic) CreatePortfolio(in *portfolio.CreatePortfolioReq) (*portfolio.PortfolioResp, error) {
	item, err := l.svcCtx.Service.CreatePortfolio(l.ctx, in)
	if err != nil {
		return nil, err
	}
	return &portfolio.PortfolioResp{Portfolio: item}, nil
}
