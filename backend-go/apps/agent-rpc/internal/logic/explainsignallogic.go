package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExplainSignalLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewExplainSignalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExplainSignalLogic {
	return &ExplainSignalLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ExplainSignalLogic) ExplainSignal(in *agent.ExplainSignalReq) (*agent.ExplainSignalResp, error) {
	result, err := l.svcCtx.Orchestrator.AnalyzeStock(l.ctx, domain.AnalyzeRequest{UserID: in.GetUserId(), TaskType: domain.TaskExplainSignal, Symbol: in.GetSymbol(), Question: in.GetQuestion(), KnowledgeBaseID: in.GetKnowledgeBaseId(), Title: "Explain Signal"})
	if err != nil {
		return nil, err
	}
	return &agent.ExplainSignalResp{Result: result}, nil
}
