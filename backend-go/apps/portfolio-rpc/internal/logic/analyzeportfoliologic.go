package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type AnalyzePortfolioLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAnalyzePortfolioLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnalyzePortfolioLogic {
	return &AnalyzePortfolioLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AnalyzePortfolioLogic) AnalyzePortfolio(in *portfolio.AnalyzePortfolioReq) (*portfolio.AnalyzePortfolioResp, error) {
	var item *portfolio.PortfolioAnalysis
	var err error
	if in.GetRefresh() {
		item, err = l.svcCtx.Service.Analyze(l.ctx, in.GetUserId(), in.GetPortfolioId())
	} else {
		item, err = l.svcCtx.Service.LatestAnalysis(l.ctx, in.GetUserId(), in.GetPortfolioId())
	}
	if err != nil {
		return nil, err
	}
	return &portfolio.AnalyzePortfolioResp{Analysis: item}, nil
}
