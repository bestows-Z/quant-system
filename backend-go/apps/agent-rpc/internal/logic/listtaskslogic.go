package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTasksLogic {
	return &ListTasksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListTasksLogic) ListTasks(in *agent.ListAgentTasksReq) (*agent.ListAgentTasksResp, error) {
	items, err := l.svcCtx.Orchestrator.ListTasks(l.ctx, in.GetUserId(), in.GetStatus(), in.GetTaskType(), in.GetLimit())
	if err != nil {
		return nil, err
	}
	return &agent.ListAgentTasksResp{Items: items}, nil
}
