package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"
)

type SectorDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSectorDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SectorDetailLogic {
	return &SectorDetailLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *SectorDetailLogic) SectorDetail(in *feature.SectorReq) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.SectorDetail(l.ctx, in.GetCode())
	return &feature.RawReply{Json: raw}, err
}
