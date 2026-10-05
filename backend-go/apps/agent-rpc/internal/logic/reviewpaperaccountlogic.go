package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReviewPaperAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReviewPaperAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewPaperAccountLogic {
	return &ReviewPaperAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ReviewPaperAccountLogic) ReviewPaperAccount(in *agent.ReviewPaperAccountReq) (*agent.ReviewPaperAccountResp, error) {
	result, err := l.svcCtx.Orchestrator.AnalyzeStock(l.ctx, domain.AnalyzeRequest{UserID: in.GetUserId(), TaskType: domain.TaskReviewPaperAccount, Question: in.GetQuestion(), KnowledgeBaseID: in.GetKnowledgeBaseId(), AccountID: in.GetAccountId(), Title: "Review Paper Account"})
	if err != nil {
		return nil, err
	}
	return &agent.ReviewPaperAccountResp{Result: result}, nil
}
