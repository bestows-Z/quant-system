package logic

import (
	"context"

	"quant-system/backend-go/apps/task-rpc/internal/svc"
	"quant-system/backend-go/apps/task-rpc/task"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelTaskLogic {
	return &CancelTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelTaskLogic) CancelTask(in *task.CancelTaskReq) (*task.CancelTaskResp, error) {
	return l.svcCtx.Task.CancelTask(l.ctx, in)
}
