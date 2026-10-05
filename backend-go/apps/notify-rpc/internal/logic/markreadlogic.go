package logic

import (
	"context"

	"quant-system/backend-go/apps/notify-rpc/internal/svc"
	"quant-system/backend-go/apps/notify-rpc/notify"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkReadLogic {
	return &MarkReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *MarkReadLogic) MarkRead(in *notify.MarkReadReq) (*notify.MarkReadResp, error) {
	return l.svcCtx.Notify.MarkRead(l.ctx, in)
}
