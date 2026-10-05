package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

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

func (l *GetTaskLogic) GetTask(in *agent.GetAgentTaskReq) (*agent.GetAgentTaskResp, error) {
	item, err := l.svcCtx.Orchestrator.GetTask(l.ctx, in.GetUserId(), in.GetTaskId())
	if err != nil {
		return nil, err
	}
	return &agent.GetAgentTaskResp{Item: item}, nil
}
