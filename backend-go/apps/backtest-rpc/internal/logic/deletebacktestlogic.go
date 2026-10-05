package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteBacktestLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteBacktestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBacktestLogic {
	return &DeleteBacktestLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteBacktestLogic) DeleteBacktest(in *backtest.DeleteBacktestReq) (*backtest.DeleteBacktestsResp, error) {
	affected, already, err := l.svcCtx.Backtest.DeleteTasks(l.ctx, in.GetUserId(), []uint64{in.GetTaskId()}, "")
	if err != nil {
		return nil, err
	}
	return &backtest.DeleteBacktestsResp{Affected: affected, AlreadyDeleted: already}, nil
}
