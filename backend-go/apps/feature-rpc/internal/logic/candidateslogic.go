package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CandidatesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCandidatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CandidatesLogic {
	return &CandidatesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CandidatesLogic) Candidates(in *feature.CandidateReq) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.Candidates(l.ctx, in.SortBy, in.Industry, in.MinConfidence, in.Limit, in.Symbol, in.Refresh)
	return &feature.RawReply{Json: raw}, err
}
