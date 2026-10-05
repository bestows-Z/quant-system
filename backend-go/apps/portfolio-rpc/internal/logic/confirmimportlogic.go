package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmImportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmImportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmImportLogic {
	return &ConfirmImportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ConfirmImportLogic) ConfirmImport(in *portfolio.ConfirmImportReq) (*portfolio.ConfirmImportResp, error) {
	return l.svcCtx.Service.ConfirmImport(l.ctx, in.GetUserId(), in.GetPortfolioId(), in.GetTaskId())
}
