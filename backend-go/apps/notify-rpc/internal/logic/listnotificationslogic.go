package logic

import (
	"context"

	"quant-system/backend-go/apps/notify-rpc/internal/svc"
	"quant-system/backend-go/apps/notify-rpc/notify"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListNotificationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListNotificationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListNotificationsLogic {
	return &ListNotificationsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListNotificationsLogic) ListNotifications(in *notify.ListNotificationsReq) (*notify.ListNotificationsResp, error) {
	return l.svcCtx.Notify.ListNotifications(l.ctx, in)
}
