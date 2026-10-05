package logic

import (
	"context"

	"quant-system/backend-go/apps/risk-rpc/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/risk"

	"github.com/zeromicro/go-zero/core/logx"
)

type AnalyzeBacktestRiskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAnalyzeBacktestRiskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnalyzeBacktestRiskLogic {
	return &AnalyzeBacktestRiskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AnalyzeBacktestRiskLogic) AnalyzeBacktestRisk(in *risk.AnalyzeBacktestRiskReq) (*risk.RiskAnalysisResp, error) {
	return l.svcCtx.Risk.AnalyzeBacktestRisk(l.ctx, in)
}
