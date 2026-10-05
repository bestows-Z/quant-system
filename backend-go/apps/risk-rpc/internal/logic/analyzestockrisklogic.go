package logic

import (
	"context"

	"quant-system/backend-go/apps/risk-rpc/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/risk"

	"github.com/zeromicro/go-zero/core/logx"
)

type AnalyzeStockRiskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAnalyzeStockRiskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnalyzeStockRiskLogic {
	return &AnalyzeStockRiskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AnalyzeStockRiskLogic) AnalyzeStockRisk(in *risk.AnalyzeStockRiskReq) (*risk.RiskAnalysisResp, error) {
	return l.svcCtx.Risk.AnalyzeStockRisk(l.ctx, in)
}
