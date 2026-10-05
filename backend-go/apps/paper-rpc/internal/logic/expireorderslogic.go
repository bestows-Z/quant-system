package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"
)

type ExpireOrdersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExpireOrdersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExpireOrdersLogic {
	return &ExpireOrdersLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *ExpireOrdersLogic) ExpireOrders(in *paper.ExpireOrdersReq) (*paper.ExpireOrdersResp, error) {
	count, err := l.svcCtx.Trading.ExpireOrders(l.ctx, in.GetLimit())
	return &paper.ExpireOrdersResp{ExpiredOrders: count}, err
}
