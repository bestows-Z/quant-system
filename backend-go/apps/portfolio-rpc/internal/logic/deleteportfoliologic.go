package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeletePortfolioLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeletePortfolioLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePortfolioLogic {
	return &DeletePortfolioLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeletePortfolioLogic) DeletePortfolio(in *portfolio.DeletePortfolioReq) (*portfolio.DeletePortfolioResp, error) {
	if err := l.svcCtx.Service.DeletePortfolio(l.ctx, in.GetUserId(), in.GetPortfolioId()); err != nil {
		return nil, err
	}
	return &portfolio.DeletePortfolioResp{Deleted: true}, nil
}
