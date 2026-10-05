package logic

import (
	"context"
	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"
)

type DeleteKnowledgeBaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteKnowledgeBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteKnowledgeBaseLogic {
	return &DeleteKnowledgeBaseLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *DeleteKnowledgeBaseLogic) DeleteKnowledgeBase(in *rag.DeleteKnowledgeBaseReq) (*rag.DeleteKnowledgeBaseResp, error) {
	return l.svcCtx.Rag.DeleteKnowledgeBase(l.ctx, in.GetUserId(), in.GetId())
}
