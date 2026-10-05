package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetReportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetReportLogic {
	return &GetReportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetReportLogic) GetReport(in *agent.GetAgentReportReq) (*agent.GetAgentReportResp, error) {
	item, err := l.svcCtx.Orchestrator.GetReport(l.ctx, in.GetUserId(), in.GetReportId())
	if err != nil {
		return nil, err
	}
	return &agent.GetAgentReportResp{Item: item}, nil
}
