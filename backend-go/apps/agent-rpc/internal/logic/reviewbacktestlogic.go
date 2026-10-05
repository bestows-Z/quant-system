package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReviewBacktestLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReviewBacktestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewBacktestLogic {
	return &ReviewBacktestLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ReviewBacktestLogic) ReviewBacktest(in *agent.ReviewBacktestReq) (*agent.ReviewBacktestResp, error) {
	result, err := l.svcCtx.Orchestrator.AnalyzeStock(l.ctx, domain.AnalyzeRequest{UserID: in.GetUserId(), TaskType: domain.TaskReviewBacktest, Question: in.GetQuestion(), KnowledgeBaseID: in.GetKnowledgeBaseId(), Title: "Review Backtest", Extra: map[string]any{"backtest_id": in.GetBacktestId()}})
	if err != nil {
		return nil, err
	}
	return &agent.ReviewBacktestResp{Result: result}, nil
}
