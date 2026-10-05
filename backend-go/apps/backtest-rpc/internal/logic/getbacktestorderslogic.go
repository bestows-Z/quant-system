package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBacktestOrdersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBacktestOrdersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBacktestOrdersLogic {
	return &GetBacktestOrdersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetBacktestOrdersLogic) GetBacktestOrders(in *backtest.GetBacktestOrdersReq) (*backtest.GetBacktestOrdersResp, error) {
	items, err := l.svcCtx.Backtest.GetOrders(l.ctx, in.GetUserId(), in.GetTaskId())
	if err != nil {
		return nil, err
	}
	return &backtest.GetBacktestOrdersResp{Items: items}, nil
}
