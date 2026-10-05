package logic

import (
	"context"

	"quant-system/backend-go/apps/model-gateway-rpc/internal/svc"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgateway"

	"github.com/zeromicro/go-zero/core/logx"
)

type TestConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTestConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestConfigLogic {
	return &TestConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TestConfigLogic) TestConfig(in *modelgateway.ConfigReq) (*modelgateway.TestConfigResp, error) {
	return l.svcCtx.Service.Test(l.ctx, in.GetUserId(), in.GetId())
}
