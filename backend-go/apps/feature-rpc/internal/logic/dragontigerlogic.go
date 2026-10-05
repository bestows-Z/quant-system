package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DragonTigerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDragonTigerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DragonTigerLogic {
	return &DragonTigerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DragonTigerLogic) DragonTiger(in *feature.DragonTigerReq) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.DragonTiger(l.ctx, in.TradeDate, in.Limit)
	return &feature.RawReply{Json: raw}, err
}
