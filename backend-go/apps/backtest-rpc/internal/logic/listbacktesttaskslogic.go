package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListBacktestTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListBacktestTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBacktestTasksLogic {
	return &ListBacktestTasksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListBacktestTasksLogic) ListBacktestTasks(in *backtest.ListBacktestTasksReq) (*backtest.ListBacktestTasksResp, error) {
	items, err := l.svcCtx.Backtest.ListTasks(l.ctx, in.GetUserId(), in.GetStatus(), in.GetLimit())
	if err != nil {
		return nil, err
	}
	return &backtest.ListBacktestTasksResp{Items: items}, nil
}
