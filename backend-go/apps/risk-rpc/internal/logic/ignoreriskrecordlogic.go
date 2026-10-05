package logic

import (
	"context"

	"quant-system/backend-go/apps/risk-rpc/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/risk"

	"github.com/zeromicro/go-zero/core/logx"
)

type IgnoreRiskRecordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIgnoreRiskRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IgnoreRiskRecordLogic {
	return &IgnoreRiskRecordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *IgnoreRiskRecordLogic) IgnoreRiskRecord(in *risk.IgnoreRiskRecordReq) (*risk.IgnoreRiskRecordResp, error) {
	return l.svcCtx.Risk.IgnoreRiskRecord(l.ctx, in)
}
