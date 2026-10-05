package logic

import (
	"context"

	"quant-system/backend-go/apps/risk-rpc/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/risk"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRiskRecordsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListRiskRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRiskRecordsLogic {
	return &ListRiskRecordsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListRiskRecordsLogic) ListRiskRecords(in *risk.ListRiskRecordsReq) (*risk.ListRiskRecordsResp, error) {
	return l.svcCtx.Risk.ListRiskRecords(l.ctx, in)
}
