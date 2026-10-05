package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOrdersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListOrdersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrdersLogic {
	return &ListOrdersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListOrdersLogic) ListOrders(in *paper.ListOrdersReq) (*paper.ListOrdersResp, error) {
	items, err := l.svcCtx.Trading.ListOrders(l.ctx, in.GetUserId(), in.GetAccountId(), in.GetSymbol(), in.GetStatus())
	if err != nil {
		return nil, err
	}
	return &paper.ListOrdersResp{Items: items}, nil
}
