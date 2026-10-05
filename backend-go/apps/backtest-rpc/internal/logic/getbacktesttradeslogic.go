package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBacktestTradesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBacktestTradesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBacktestTradesLogic {
	return &GetBacktestTradesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetBacktestTradesLogic) GetBacktestTrades(in *backtest.GetBacktestTradesReq) (*backtest.GetBacktestTradesResp, error) {
	items, err := l.svcCtx.Backtest.GetTrades(l.ctx, in.GetUserId(), in.GetTaskId())
	if err != nil {
		return nil, err
	}
	return &backtest.GetBacktestTradesResp{Items: items}, nil
}
