package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBacktestResultLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBacktestResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBacktestResultLogic {
	return &GetBacktestResultLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetBacktestResultLogic) GetBacktestResult(in *backtest.GetBacktestResultReq) (*backtest.GetBacktestResultResp, error) {
	item, err := l.svcCtx.Backtest.GetResult(l.ctx, in.GetUserId(), in.GetTaskId())
	if err != nil {
		return nil, err
	}
	return &backtest.GetBacktestResultResp{Item: item}, nil
}
