package logic

import (
	"context"

	"quant-system/backend-go/apps/model-gateway-rpc/internal/svc"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"

	"github.com/zeromicro/go-zero/core/logx"
)

type ActivateConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewActivateConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ActivateConfigLogic {
	return &ActivateConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ActivateConfigLogic) ActivateConfig(in *modelgateway.ConfigReq) (*modelgateway.ActivateConfigResp, error) {
	item, err := l.svcCtx.Service.Activate(l.ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	return &modelgateway.ActivateConfigResp{Config: item}, nil
}
