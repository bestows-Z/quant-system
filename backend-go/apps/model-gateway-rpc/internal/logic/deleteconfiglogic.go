package logic

import (
	"context"

	"quant-system/backend-go/apps/model-gateway-rpc/internal/svc"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConfigLogic {
	return &DeleteConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteConfigLogic) DeleteConfig(in *modelgateway.ConfigReq) (*modelgateway.DeleteConfigResp, error) {
	if err := l.svcCtx.Service.Delete(l.ctx, in.GetUserId(), in.GetId()); err != nil {
		return nil, err
	}
	return &modelgateway.DeleteConfigResp{Deleted: true}, nil
}
