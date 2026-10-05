package logic

import (
	"context"

	"quant-system/backend-go/apps/task-rpc/internal/svc"
	"quant-system/backend-go/apps/task-rpc/task"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTaskLogsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTaskLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTaskLogsLogic {
	return &ListTaskLogsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListTaskLogsLogic) ListTaskLogs(in *task.ListTaskLogsReq) (*task.ListTaskLogsResp, error) {
	return l.svcCtx.Task.ListTaskLogs(l.ctx, in)
}
