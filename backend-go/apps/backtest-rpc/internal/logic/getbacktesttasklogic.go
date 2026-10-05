package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBacktestTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBacktestTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBacktestTaskLogic {
	return &GetBacktestTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetBacktestTaskLogic) GetBacktestTask(in *backtest.GetBacktestTaskReq) (*backtest.GetBacktestTaskResp, error) {
	item, err := l.svcCtx.Backtest.GetTask(l.ctx, in.GetUserId(), in.GetTaskId())
	if err != nil {
		return nil, err
	}
	return &backtest.GetBacktestTaskResp{Item: item}, nil
}
