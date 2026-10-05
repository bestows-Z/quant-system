package logic

import (
	"context"

	"quant-system/backend-go/apps/model-gateway-rpc/internal/svc"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListConfigsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListConfigsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConfigsLogic {
	return &ListConfigsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListConfigsLogic) ListConfigs(in *modelgateway.ListConfigsReq) (*modelgateway.ListConfigsResp, error) {
	items, err := l.svcCtx.Service.List(l.ctx, in.GetUserId())
	if err != nil {
		return nil, err
	}
	return &modelgateway.ListConfigsResp{Items: items}, nil
}
