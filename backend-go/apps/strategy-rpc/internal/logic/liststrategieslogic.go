package logic

import (
	"context"

	"quant-system/backend-go/apps/strategy-rpc/internal/svc"
	"quant-system/backend-go/apps/strategy-rpc/strategy"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListStrategiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListStrategiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListStrategiesLogic {
	return &ListStrategiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListStrategiesLogic) ListStrategies(in *strategy.ListStrategiesReq) (*strategy.ListStrategiesResp, error) {
	items, err := l.svcCtx.Strategy.ListStrategies(l.ctx, in.GetUserId())
	if err != nil {
		return nil, err
	}
	return &strategy.ListStrategiesResp{Items: items}, nil
}
