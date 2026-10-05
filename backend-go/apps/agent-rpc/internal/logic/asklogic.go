package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AskLogic {
	return &AskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AskLogic) Ask(in *agent.AgentAskReq) (*agent.AgentAskResp, error) {
	result, err := l.svcCtx.Orchestrator.Ask(l.ctx, domain.AnalyzeRequest{
		UserID:          in.GetUserId(),
		Symbol:          in.GetSymbol(),
		Question:        in.GetQuestion(),
		KnowledgeBaseID: in.GetKnowledgeBaseId(),
		AccountID:       in.GetAccountId(),
	})
	if err != nil {
		return nil, err
	}
	return &agent.AgentAskResp{Result: result}, nil
}
