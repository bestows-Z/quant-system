package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderLogic {
	return &GetOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetOrderLogic) GetOrder(in *paper.GetOrderReq) (*paper.GetOrderResp, error) {
	order, err := l.svcCtx.Trading.GetOrder(l.ctx, in.GetUserId(), in.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &paper.GetOrderResp{Order: order}, nil
}
