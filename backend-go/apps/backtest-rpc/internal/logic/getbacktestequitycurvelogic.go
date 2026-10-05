package logic

import (
	"context"

	"quant-system/backend-go/apps/backtest-rpc/backtest"
	"quant-system/backend-go/apps/backtest-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBacktestEquityCurveLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBacktestEquityCurveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBacktestEquityCurveLogic {
	return &GetBacktestEquityCurveLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetBacktestEquityCurveLogic) GetBacktestEquityCurve(in *backtest.GetBacktestEquityCurveReq) (*backtest.GetBacktestEquityCurveResp, error) {
	items, err := l.svcCtx.Backtest.GetCurve(l.ctx, in.GetUserId(), in.GetTaskId())
	if err != nil {
		return nil, err
	}
	return &backtest.GetBacktestEquityCurveResp{Items: items}, nil
}
