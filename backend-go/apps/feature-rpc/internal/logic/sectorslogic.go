package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SectorsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSectorsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SectorsLogic {
	return &SectorsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SectorsLogic) Sectors(in *feature.SectorListReq) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.Sectors(l.ctx, in.GetType(), in.GetSortBy(), in.GetOrder())
	return &feature.RawReply{Json: raw}, err
}
