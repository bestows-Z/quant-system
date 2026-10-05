package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListReportsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListReportsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListReportsLogic {
	return &ListReportsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListReportsLogic) ListReports(in *agent.ListAgentReportsReq) (*agent.ListAgentReportsResp, error) {
	items, total, err := l.svcCtx.Orchestrator.ListReports(l.ctx, in)
	if err != nil {
		return nil, err
	}
	return &agent.ListAgentReportsResp{Items: items, Total: total}, nil
}
