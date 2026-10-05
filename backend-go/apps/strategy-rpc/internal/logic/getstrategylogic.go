package logic

import (
	"context"

	"quant-system/backend-go/apps/strategy-rpc/internal/svc"
	"quant-system/backend-go/apps/strategy-rpc/strategy"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetStrategyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetStrategyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetStrategyLogic {
	return &GetStrategyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetStrategyLogic) GetStrategy(in *strategy.GetStrategyReq) (*strategy.GetStrategyResp, error) {
	item, err := l.svcCtx.Strategy.GetStrategy(l.ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	return &strategy.GetStrategyResp{Item: item}, nil
}
