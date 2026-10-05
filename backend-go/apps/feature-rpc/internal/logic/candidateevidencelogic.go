package logic

import (
	"context"

	"quant-system/backend-go/apps/feature-rpc/feature"
	"quant-system/backend-go/apps/feature-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CandidateEvidenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCandidateEvidenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CandidateEvidenceLogic {
	return &CandidateEvidenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CandidateEvidenceLogic) CandidateEvidence(in *feature.CandidateReq) (*feature.RawReply, error) {
	raw, err := l.svcCtx.Service.Candidates(l.ctx, in.SortBy, in.Industry, in.MinConfidence, 1, in.Symbol, false)
	return &feature.RawReply{Json: raw}, err
}
