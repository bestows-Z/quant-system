package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAccountAssetsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAccountAssetsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountAssetsLogic {
	return &GetAccountAssetsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAccountAssetsLogic) GetAccountAssets(in *paper.GetAccountAssetsReq) (*paper.GetAccountAssetsResp, error) {
	account, err := l.svcCtx.Trading.GetAccount(l.ctx, in.GetUserId(), in.GetAccountId())
	if err != nil {
		return nil, err
	}
	return &paper.GetAccountAssetsResp{Account: account}, nil
}
