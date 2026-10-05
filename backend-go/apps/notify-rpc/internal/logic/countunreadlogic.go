package logic

import (
	"context"

	"quant-system/backend-go/apps/notify-rpc/internal/svc"
	"quant-system/backend-go/apps/notify-rpc/notify"

	"github.com/zeromicro/go-zero/core/logx"
)

type CountUnreadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCountUnreadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CountUnreadLogic {
	return &CountUnreadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CountUnreadLogic) CountUnread(in *notify.CountUnreadReq) (*notify.CountUnreadResp, error) {
	return l.svcCtx.Notify.CountUnread(l.ctx, in)
}
