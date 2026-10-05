package logic

import (
	"context"

	"quant-system/backend-go/apps/model-gateway-rpc/internal/svc"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListProvidersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListProvidersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProvidersLogic {
	return &ListProvidersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListProvidersLogic) ListProviders(in *modelgateway.ListProvidersReq) (*modelgateway.ListProvidersResp, error) {
	items, err := l.svcCtx.Service.Providers(l.ctx)
	if err != nil {
		return nil, err
	}
	return &modelgateway.ListProvidersResp{Items: items}, nil
}
