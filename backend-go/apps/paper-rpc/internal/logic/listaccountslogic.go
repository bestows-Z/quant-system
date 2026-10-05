package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAccountsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAccountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAccountsLogic {
	return &ListAccountsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAccountsLogic) ListAccounts(in *paper.ListAccountsReq) (*paper.ListAccountsResp, error) {
	items, err := l.svcCtx.Trading.ListAccounts(l.ctx, in.GetUserId())
	if err != nil {
		return nil, err
	}
	return &paper.ListAccountsResp{Items: items}, nil
}
