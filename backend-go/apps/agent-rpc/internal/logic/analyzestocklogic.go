package logic

import (
	"context"

	"quant-system/backend-go/apps/agent-rpc/agent"
	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/agent-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AnalyzeStockLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAnalyzeStockLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnalyzeStockLogic {
	return &AnalyzeStockLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AnalyzeStockLogic) AnalyzeStock(in *agent.AnalyzeStockReq) (*agent.AnalyzeStockResp, error) {
	return l.svcCtx.Orchestrator.AnalyzeStock(l.ctx, domain.AnalyzeRequest{
		UserID:          in.GetUserId(),
		TaskType:        domain.TaskAnalyzeStock,
		Symbol:          in.GetSymbol(),
		Question:        in.GetQuestion(),
		KnowledgeBaseID: in.GetKnowledgeBaseId(),
		AccountID:       in.GetAccountId(),
		Title:           "Analyze Stock",
	})
}
