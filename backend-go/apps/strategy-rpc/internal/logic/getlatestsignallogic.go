package logic

import (
	"context"

	"quant-system/backend-go/apps/strategy-rpc/internal/svc"
	"quant-system/backend-go/apps/strategy-rpc/strategy"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLatestSignalLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetLatestSignalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLatestSignalLogic {
	return &GetLatestSignalLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetLatestSignalLogic) GetLatestSignal(in *strategy.GetLatestSignalReq) (*strategy.GetLatestSignalResp, error) {
	item, err := l.svcCtx.Strategy.GetLatestSignal(l.ctx, in.GetUserId(), in.GetSymbol())
	if err != nil {
		return nil, err
	}
	return &strategy.GetLatestSignalResp{Item: item}, nil
}
