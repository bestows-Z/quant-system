package logic

import (
	"context"

	"quant-system/backend-go/apps/portfolio-rpc/internal/svc"
	"quant-system/backend-go/apps/portfolio-rpc/portfolio"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewImportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPreviewImportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewImportLogic {
	return &PreviewImportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PreviewImportLogic) PreviewImport(in *portfolio.PreviewImportReq) (*portfolio.PreviewImportResp, error) {
	return l.svcCtx.Service.PreviewImport(l.ctx, in)
}
