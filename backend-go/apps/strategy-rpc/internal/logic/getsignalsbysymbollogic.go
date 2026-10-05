package logic

import (
	"context"

	"quant-system/backend-go/apps/strategy-rpc/internal/svc"
	"quant-system/backend-go/apps/strategy-rpc/strategy"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSignalsBySymbolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSignalsBySymbolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSignalsBySymbolLogic {
	return &GetSignalsBySymbolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetSignalsBySymbolLogic) GetSignalsBySymbol(in *strategy.GetSignalsBySymbolReq) (*strategy.GetSignalsBySymbolResp, error) {
	items, err := l.svcCtx.Strategy.GetSignalsBySymbol(l.ctx, in.GetUserId(), in.GetSymbol(), in.GetLimit())
	if err != nil {
		return nil, err
	}
	return &strategy.GetSignalsBySymbolResp{Items: items}, nil
}
