package logic

import (
	"context"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type TimeshareLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTimeshareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TimeshareLogic {
	return &TimeshareLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TimeshareLogic) Timeshare(in *market.SymbolRequest) (*market.RawReply, error) {
	return NewIntradayLogic(l.ctx, l.svcCtx).Timeshare(in)
}
