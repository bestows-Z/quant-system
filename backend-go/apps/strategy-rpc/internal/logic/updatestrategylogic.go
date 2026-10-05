package logic

import (
	"context"

	"quant-system/backend-go/apps/strategy-rpc/internal/svc"
	"quant-system/backend-go/apps/strategy-rpc/strategy"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateStrategyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateStrategyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateStrategyLogic {
	return &UpdateStrategyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateStrategyLogic) UpdateStrategy(in *strategy.UpdateStrategyReq) (*strategy.UpdateStrategyResp, error) {
	item, err := l.svcCtx.Strategy.UpdateStrategy(l.ctx, in)
	if err != nil {
		return nil, err
	}
	return &strategy.UpdateStrategyResp{Item: item}, nil
}
