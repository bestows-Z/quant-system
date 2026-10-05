package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RunBacktestLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunBacktestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunBacktestLogic {
	return &RunBacktestLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RunBacktestLogic) RunBacktest(in *backtest.RunBacktestReq) (*backtest.RunBacktestResp, error) {
	return l.svcCtx.Backtest.RunBacktest(l.ctx, in)
}
