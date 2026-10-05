package logic

import (
	"context"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type OrderBookLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOrderBookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OrderBookLogic {
	return &OrderBookLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *OrderBookLogic) OrderBook(in *market.SymbolRequest) (*market.RawReply, error) {
	return NewIntradayLogic(l.ctx, l.svcCtx).OrderBook(in)
}
