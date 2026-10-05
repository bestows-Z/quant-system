package logic

import (
	"context"

	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDocumentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListDocumentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDocumentsLogic {
	return &ListDocumentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListDocumentsLogic) ListDocuments(in *rag.ListDocumentsReq) (*rag.ListDocumentsResp, error) {
	items, err := l.svcCtx.Rag.ListDocuments(l.ctx, in.GetUserId(), in.GetKnowledgeBaseId())
	if err != nil {
		return nil, err
	}
	return &rag.ListDocumentsResp{Items: items}, nil
}
