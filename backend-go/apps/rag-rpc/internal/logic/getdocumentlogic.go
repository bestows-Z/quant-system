package logic

import (
	"context"

	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentLogic {
	return &GetDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDocumentLogic) GetDocument(in *rag.GetDocumentReq) (*rag.GetDocumentResp, error) {
	item, err := l.svcCtx.Rag.GetDocument(l.ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	return &rag.GetDocumentResp{Item: item}, nil
}
