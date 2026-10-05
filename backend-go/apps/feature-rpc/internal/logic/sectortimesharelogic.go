package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"
)

type SectorTimeshareLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSectorTimeshareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SectorTimeshareLogic {
	return &SectorTimeshareLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *SectorTimeshareLogic) SectorTimeshare(in *feature.SectorReq) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.SectorTimeshare(l.ctx, in.GetCode())
	return &feature.RawReply{Json: raw}, err
}
