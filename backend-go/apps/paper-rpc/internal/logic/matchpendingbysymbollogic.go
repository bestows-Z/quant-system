package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"
)

type MatchPendingBySymbolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMatchPendingBySymbolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MatchPendingBySymbolLogic {
	return &MatchPendingBySymbolLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *MatchPendingBySymbolLogic) MatchPendingBySymbol(in *paper.MatchPendingBySymbolReq) (*paper.MatchPendingBySymbolResp, error) {
	orders, trades, err := l.svcCtx.Trading.MatchPendingBySymbol(l.ctx, in.GetSymbol(), in.GetOrderBookJson())
	return &paper.MatchPendingBySymbolResp{MatchedOrders: orders, CreatedTrades: trades}, err
}
