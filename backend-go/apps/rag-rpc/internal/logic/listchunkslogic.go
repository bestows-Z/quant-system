package logic

import (
	"context"

	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListChunksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListChunksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListChunksLogic {
	return &ListChunksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListChunksLogic) ListChunks(in *rag.ListChunksReq) (*rag.ListChunksResp, error) {
	items, err := l.svcCtx.Rag.ListChunks(l.ctx, in.GetUserId(), in.GetDocumentId())
	if err != nil {
		return nil, err
	}
	return &rag.ListChunksResp{Items: items}, nil
}
