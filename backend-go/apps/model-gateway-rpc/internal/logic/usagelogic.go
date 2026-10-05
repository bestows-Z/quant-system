package logic

import (
	"context"

	"quant-system/backend-go/apps/model-gateway-rpc/internal/svc"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"

	"github.com/zeromicro/go-zero/core/logx"
)

type UsageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUsageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UsageLogic {
	return &UsageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UsageLogic) Usage(in *modelgateway.UsageReq) (*modelgateway.UsageSummary, error) {
	return l.svcCtx.Service.Usage(l.ctx, in.GetUserId(), in.GetDays())
}
