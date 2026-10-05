package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPositionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPositionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPositionsLogic {
	return &ListPositionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListPositionsLogic) ListPositions(in *paper.ListPositionsReq) (*paper.ListPositionsResp, error) {
	items, err := l.svcCtx.Trading.ListPositions(l.ctx, in.GetUserId(), in.GetAccountId())
	if err != nil {
		return nil, err
	}
	return &paper.ListPositionsResp{Items: items}, nil
}
