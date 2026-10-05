package logic

import (
	"context"

	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKnowledgeBasesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKnowledgeBasesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKnowledgeBasesLogic {
	return &ListKnowledgeBasesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListKnowledgeBasesLogic) ListKnowledgeBases(in *rag.ListKnowledgeBasesReq) (*rag.ListKnowledgeBasesResp, error) {
	items, err := l.svcCtx.Rag.ListKnowledgeBases(l.ctx, in.GetUserId())
	if err != nil {
		return nil, err
	}
	return &rag.ListKnowledgeBasesResp{Items: items}, nil
}
