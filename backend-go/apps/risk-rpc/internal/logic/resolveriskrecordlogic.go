package logic

import (
	"context"

	"quant-system/backend-go/apps/risk-rpc/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/risk"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResolveRiskRecordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResolveRiskRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResolveRiskRecordLogic {
	return &ResolveRiskRecordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ResolveRiskRecordLogic) ResolveRiskRecord(in *risk.ResolveRiskRecordReq) (*risk.ResolveRiskRecordResp, error) {
	return l.svcCtx.Risk.ResolveRiskRecord(l.ctx, in)
}
