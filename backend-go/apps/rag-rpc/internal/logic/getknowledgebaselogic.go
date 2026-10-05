package logic

import (
	"context"

	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetKnowledgeBaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetKnowledgeBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKnowledgeBaseLogic {
	return &GetKnowledgeBaseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetKnowledgeBaseLogic) GetKnowledgeBase(in *rag.GetKnowledgeBaseReq) (*rag.GetKnowledgeBaseResp, error) {
	item, err := l.svcCtx.Rag.GetKnowledgeBase(l.ctx, in.GetUserId(), in.GetId())
	if err != nil {
		return nil, err
	}
	return &rag.GetKnowledgeBaseResp{Item: item}, nil
}
