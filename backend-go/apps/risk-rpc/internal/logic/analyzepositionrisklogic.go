package logic

import (
	"context"

	"quant-system/backend-go/apps/risk-rpc/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/risk"

	"github.com/zeromicro/go-zero/core/logx"
)

type AnalyzePositionRiskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAnalyzePositionRiskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnalyzePositionRiskLogic {
	return &AnalyzePositionRiskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AnalyzePositionRiskLogic) AnalyzePositionRisk(in *risk.AnalyzePositionRiskReq) (*risk.RiskAnalysisResp, error) {
	return l.svcCtx.Risk.AnalyzePositionRisk(l.ctx, in)
}
