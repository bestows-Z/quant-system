package logic

import (
	"context"

	"quant-system/backend-go/apps/risk-rpc/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/risk"

	"github.com/zeromicro/go-zero/core/logx"
)

type AnalyzeAccountRiskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAnalyzeAccountRiskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnalyzeAccountRiskLogic {
	return &AnalyzeAccountRiskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AnalyzeAccountRiskLogic) AnalyzeAccountRisk(in *risk.AnalyzeAccountRiskReq) (*risk.RiskAnalysisResp, error) {
	return l.svcCtx.Risk.AnalyzeAccountRisk(l.ctx, in)
}
