package logic

import (
	"context"

	"quant-system/backend-go/apps/paper-rpc/internal/svc"
	"quant-system/backend-go/apps/paper-rpc/paper"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEquityCurveLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetEquityCurveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEquityCurveLogic {
	return &GetEquityCurveLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetEquityCurveLogic) GetEquityCurve(in *paper.GetEquityCurveReq) (*paper.GetEquityCurveResp, error) {
	items, err := l.svcCtx.Trading.EquityCurve(l.ctx, in.GetUserId(), in.GetAccountId())
	if err != nil {
		return nil, err
	}
	return &paper.GetEquityCurveResp{Items: items}, nil
}
