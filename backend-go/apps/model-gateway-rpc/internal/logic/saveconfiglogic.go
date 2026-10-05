package logic

import (
	"context"

	"quant-system/backend-go/apps/model-gateway-rpc/internal/svc"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSaveConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveConfigLogic {
	return &SaveConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SaveConfigLogic) SaveConfig(in *modelgateway.SaveConfigReq) (*modelgateway.ConfigResp, error) {
	item, err := l.svcCtx.Service.Save(l.ctx, in)
	if err != nil {
		return nil, err
	}
	return &modelgateway.ConfigResp{Config: item}, nil
}
