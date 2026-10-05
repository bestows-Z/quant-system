package logic

import (
	"context"
	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"
)

type DeleteDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDocumentLogic {
	return &DeleteDocumentLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *DeleteDocumentLogic) DeleteDocument(in *rag.DeleteDocumentReq) (*rag.DeleteDocumentResp, error) {
	return l.svcCtx.Rag.DeleteDocument(l.ctx, in.GetUserId(), in.GetId())
}
