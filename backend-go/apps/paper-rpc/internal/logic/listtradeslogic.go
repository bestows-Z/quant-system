package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTradesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTradesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTradesLogic {
	return &ListTradesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListTradesLogic) ListTrades(in *paper.ListTradesReq) (*paper.ListTradesResp, error) {
	items, err := l.svcCtx.Trading.ListTrades(l.ctx, in.GetUserId(), in.GetAccountId(), in.GetSymbol())
	if err != nil {
		return nil, err
	}
	return &paper.ListTradesResp{Items: items}, nil
}
