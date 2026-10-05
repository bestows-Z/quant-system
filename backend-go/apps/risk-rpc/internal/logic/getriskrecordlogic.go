package logic

import (
	"context"

	"quant-system/backend-go/apps/risk-rpc/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/risk"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRiskRecordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRiskRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRiskRecordLogic {
	return &GetRiskRecordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetRiskRecordLogic) GetRiskRecord(in *risk.GetRiskRecordReq) (*risk.GetRiskRecordResp, error) {
	return l.svcCtx.Risk.GetRiskRecord(l.ctx, in)
}
