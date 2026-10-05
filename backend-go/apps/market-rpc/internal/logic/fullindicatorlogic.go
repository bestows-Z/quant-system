package logic

import (
	"context"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type FullIndicatorLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFullIndicatorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FullIndicatorLogic {
	return &FullIndicatorLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FullIndicatorLogic) FullIndicator(in *market.IndicatorRequest) (*market.RawReply, error) {
	adjust := valueOr(in.GetAdjust(), "qfq")
	if !requestNeedsRealtimeBar(in.GetEndDate(), time.Now()) {
		if reply := l.databaseFullIndicator(in, adjust); reply != nil {
			return reply, nil
		}
	}
	jsonText, err := l.svcCtx.QuantPython.FullIndicator(l.ctx, in.GetSymbol(), in.GetPeriod(), in.GetStartDate(), in.GetEndDate(), adjust)
	if err == nil && rawSuccess(jsonText) {
		return &market.RawReply{Json: jsonText}, nil
	}
	if reply := l.databaseFullIndicator(in, adjust); reply != nil {
		return reply, nil
	}
	if err != nil {
		return nil, err
	}
	return &market.RawReply{Json: jsonText}, nil
}

func (l *FullIndicatorLogic) databaseFullIndicator(in *market.IndicatorRequest, adjust string) *market.RawReply {
	if bars := loadBarsWithRealtimeSnapshot(l.ctx, l.svcCtx, in.GetSymbol(), in.GetStartDate(), in.GetEndDate(), adjust); len(bars) > 0 {
		source := barStorageSource(bars)
		payload := map[string]any{
			"symbol":     in.GetSymbol(),
			"indicators": fullIndicatorsFromBars(bars),
			"source":     source,
			"data_mode":  modeForSource(source),
		}
		return &market.RawReply{Json: successEnvelope(payload)}
	}
	return nil
}
