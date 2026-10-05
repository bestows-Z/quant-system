package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"
)

type SectorConstituentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSectorConstituentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SectorConstituentsLogic {
	return &SectorConstituentsLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *SectorConstituentsLogic) SectorConstituents(in *feature.SectorReq) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.SectorConstituents(l.ctx, in.GetCode(), in.GetSortBy(), in.GetOrder(), in.GetLimit())
	return &feature.RawReply{Json: raw}, err
}
