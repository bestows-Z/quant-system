package logic

import (
	"context"

	"quant-system/backend-go/apps/task-rpc/internal/svc"
	"quant-system/backend-go/apps/task-rpc/task"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddTaskLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddTaskLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddTaskLogLogic {
	return &AddTaskLogLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AddTaskLogLogic) AddTaskLog(in *task.AddTaskLogReq) (*task.AddTaskLogResp, error) {
	return l.svcCtx.Task.AddTaskLog(l.ctx, in)
}
