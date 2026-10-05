package logic

import (
	"context"

	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDocumentLogic {
	return &CreateDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateDocumentLogic) CreateDocument(in *rag.CreateDocumentReq) (*rag.CreateDocumentResp, error) {
	item, err := l.svcCtx.Rag.CreateDocument(l.ctx, in)
	if err != nil {
		return nil, err
	}
	return &rag.CreateDocumentResp{Item: item}, nil
}
