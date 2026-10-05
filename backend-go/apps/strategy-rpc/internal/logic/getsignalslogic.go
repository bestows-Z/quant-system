package logic

import (
	"context"

	"quant-system/backend-go/apps/strategy-rpc/internal/svc"
	"quant-system/backend-go/apps/strategy-rpc/strategy"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSignalsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSignalsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSignalsLogic {
	return &GetSignalsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetSignalsLogic) GetSignals(in *strategy.GetSignalsReq) (*strategy.GetSignalsResp, error) {
	items, err := l.svcCtx.Strategy.GetSignals(l.ctx, in.GetUserId(), in.GetStrategyCode(), in.GetSignalType(), in.GetLimit())
	if err != nil {
		return nil, err
	}
	return &strategy.GetSignalsResp{Items: items}, nil
}
