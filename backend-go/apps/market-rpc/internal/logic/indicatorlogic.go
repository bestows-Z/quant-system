package logic

import (
	"context"
	"strings"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type IndicatorLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIndicatorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IndicatorLogic {
	return &IndicatorLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *IndicatorLogic) Indicator(in *market.IndicatorRequest) (*market.RawReply, error) {
	adjust := valueOr(in.GetAdjust(), "qfq")
	if !requestNeedsRealtimeBar(in.GetEndDate(), time.Now()) {
		if reply := l.databaseIndicator(in, adjust); reply != nil {
			return reply, nil
		}
	}
	jsonText, err := l.svcCtx.QuantPython.Indicator(l.ctx, in.GetSymbol(), in.GetType(), in.GetPeriod(), in.GetStartDate(), in.GetEndDate(), adjust)
	if err == nil && rawSuccess(jsonText) {
		return &market.RawReply{Json: jsonText}, nil
	}
	if reply := l.databaseIndicator(in, adjust); reply != nil {
		return reply, nil
	}
	if err != nil {
		return nil, err
	}
	return &market.RawReply{Json: jsonText}, nil
}

func (l *IndicatorLogic) databaseIndicator(in *market.IndicatorRequest, adjust string) *market.RawReply {
	if bars := loadBarsWithRealtimeSnapshot(l.ctx, l.svcCtx, in.GetSymbol(), in.GetStartDate(), in.GetEndDate(), adjust); len(bars) > 0 {
		source := barStorageSource(bars)
		all := fullIndicatorsFromBars(bars)
		indicatorType := strings.ToLower(valueOr(in.GetType(), "macd"))
		payload := map[string]any{
			"symbol":         in.GetSymbol(),
			"indicator_type": indicatorType,
			"items":          all[indicatorType],
			"count":          0,
			"source":         source,
			"data_mode":      modeForSource(source),
		}
		if items, ok := all[indicatorType].([]map[string]any); ok {
			payload["count"] = len(items)
		}
		return &market.RawReply{Json: successEnvelope(payload)}
	}
	return nil
}
