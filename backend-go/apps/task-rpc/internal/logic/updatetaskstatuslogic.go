package logic

import (
	"context"

	"quant-system/backend-go/apps/task-rpc/internal/svc"
	"quant-system/backend-go/apps/task-rpc/task"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTaskStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateTaskStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTaskStatusLogic {
	return &UpdateTaskStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateTaskStatusLogic) UpdateTaskStatus(in *task.UpdateTaskStatusReq) (*task.UpdateTaskStatusResp, error) {
	return l.svcCtx.Task.UpdateTaskStatus(l.ctx, in)
}
