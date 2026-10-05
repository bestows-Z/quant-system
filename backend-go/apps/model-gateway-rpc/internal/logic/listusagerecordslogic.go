package logic

import (
	"context"

	"quant-system/backend-go/apps/model-gateway-rpc/internal/svc"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListUsageRecordsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListUsageRecordsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUsageRecordsLogic {
	return &ListUsageRecordsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListUsageRecordsLogic) ListUsageRecords(in *modelgateway.ListUsageRecordsReq) (*modelgateway.ListUsageRecordsResp, error) {
	items, total, err := l.svcCtx.Service.UsageRecords(l.ctx, in)
	if err != nil {
		return nil, err
	}
	return &modelgateway.ListUsageRecordsResp{Items: items, Total: total}, nil
}
