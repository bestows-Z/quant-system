package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelOrderLogic {
	return &CancelOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelOrderLogic) CancelOrder(in *paper.CancelOrderReq) (*paper.CancelOrderResp, error) {
	order, err := l.svcCtx.Trading.CancelOrder(l.ctx, in.GetUserId(), in.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &paper.CancelOrderResp{Order: order}, nil
}
