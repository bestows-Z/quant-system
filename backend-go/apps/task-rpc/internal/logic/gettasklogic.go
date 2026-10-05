package logic

import (
	"context"

	"quant-system/backend-go/apps/task-rpc/internal/svc"
	"quant-system/backend-go/apps/task-rpc/task"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskLogic {
	return &GetTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetTaskLogic) GetTask(in *task.GetTaskReq) (*task.GetTaskResp, error) {
	return l.svcCtx.Task.GetTask(l.ctx, in)
}
