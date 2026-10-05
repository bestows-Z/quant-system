package logic

import (
	"context"

	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateKnowledgeBaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateKnowledgeBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateKnowledgeBaseLogic {
	return &CreateKnowledgeBaseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateKnowledgeBaseLogic) CreateKnowledgeBase(in *rag.CreateKnowledgeBaseReq) (*rag.CreateKnowledgeBaseResp, error) {
	item, err := l.svcCtx.Rag.CreateKnowledgeBase(l.ctx, in)
	if err != nil {
		return nil, err
	}
	return &rag.CreateKnowledgeBaseResp{Item: item}, nil
}
