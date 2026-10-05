package logic

import (
	"context"

	"quant-system/backend-go/apps/rag-rpc/internal/svc"
	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
)

type AskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AskLogic {
	return &AskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AskLogic) Ask(in *rag.AskReq) (*rag.AskResp, error) {
	return l.svcCtx.Rag.Ask(l.ctx, in)
}
