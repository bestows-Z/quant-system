package logic

import (
	"context"

	"quant-system/backend-go/apps/notify-rpc/internal/svc"
	"quant-system/backend-go/apps/notify-rpc/notify"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteNotificationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteNotificationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteNotificationLogic {
	return &DeleteNotificationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteNotificationLogic) DeleteNotification(in *notify.DeleteNotificationReq) (*notify.DeleteNotificationResp, error) {
	return l.svcCtx.Notify.DeleteNotification(l.ctx, in)
}
