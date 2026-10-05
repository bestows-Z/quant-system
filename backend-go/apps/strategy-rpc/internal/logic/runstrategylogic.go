package logic

import (
	"context"

	"quant-system/backend-go/apps/strategy-rpc/internal/svc"
	"quant-system/backend-go/apps/strategy-rpc/strategy"

	"github.com/zeromicro/go-zero/core/logx"
)

type RunStrategyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunStrategyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunStrategyLogic {
	return &RunStrategyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RunStrategyLogic) RunStrategy(in *strategy.RunStrategyReq) (*strategy.RunStrategyResp, error) {
	return l.svcCtx.Strategy.RunStrategy(l.ctx, in)
}
