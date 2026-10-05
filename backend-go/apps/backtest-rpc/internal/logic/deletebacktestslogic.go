package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteBacktestsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteBacktestsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBacktestsLogic {
	return &DeleteBacktestsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteBacktestsLogic) DeleteBacktests(in *backtest.DeleteBacktestsReq) (*backtest.DeleteBacktestsResp, error) {
	affected, already, err := l.svcCtx.Backtest.DeleteTasks(l.ctx, in.GetUserId(), in.GetTaskIds(), in.GetScope())
	if err != nil {
		return nil, err
	}
	return &backtest.DeleteBacktestsResp{Affected: affected, AlreadyDeleted: already}, nil
}
