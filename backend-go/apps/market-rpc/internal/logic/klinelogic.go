package logic

import (
	"context"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type KlineLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKlineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KlineLogic {
	return &KlineLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *KlineLogic) Kline(in *market.KlineRequest) (*market.RawReply, error) {
	adjust := valueOr(in.GetAdjust(), "qfq")
	if !requestNeedsRealtimeBar(in.GetEndDate(), time.Now()) {
		if bars, err := l.svcCtx.Market.ListBars(l.ctx, in.GetSymbol(), in.GetStartDate(), in.GetEndDate(), adjust); err == nil && len(bars) > 0 {
			return &market.RawReply{Json: successEnvelope(klinePayload(in.GetSymbol(), in.GetPeriod(), bars, barStorageSource(bars)))}, nil
		}
	}
	jsonText, err := l.svcCtx.QuantPython.Kline(l.ctx, in.GetSymbol(), in.GetPeriod(), in.GetStartDate(), in.GetEndDate(), adjust)
	if err != nil {
		return l.klineFallback(in, adjust, "", err)
	}
	if rawSuccess(jsonText) {
		_ = l.svcCtx.Market.UpsertBars(l.ctx, parseBars(rawData(jsonText), adjust))
		return &market.RawReply{Json: jsonText}, nil
	}
	return l.klineFallback(in, adjust, jsonText, nil)
}

func (l *KlineLogic) klineFallback(in *market.KlineRequest, adjust, sourceError string, err error) (*market.RawReply, error) {
	if bars := loadBarsWithRealtimeSnapshot(l.ctx, l.svcCtx, in.GetSymbol(), in.GetStartDate(), in.GetEndDate(), adjust); len(bars) > 0 {
		if barStorageSource(bars) == "local_csv" {
			payload := klinePayload(in.GetSymbol(), in.GetPeriod(), bars, "local_csv")
			payload["message"] = "本地初始化历史数据，非实时行情"
			return &market.RawReply{Json: successEnvelope(payload)}, nil
		}
		payload := klinePayload(in.GetSymbol(), in.GetPeriod(), bars, "partial_database")
		payload["message"] = "行情源不可用，当前为本地部分历史数据"
		return &market.RawReply{Json: successEnvelope(payload)}, nil
	}
	if sourceError != "" {
		return &market.RawReply{Json: sourceError}, nil
	}
	message := "source_unavailable"
	if err != nil {
		message = err.Error()
	}
	return &market.RawReply{Json: unavailableEnvelope("source_unavailable: 所有真实行情源不可用", []map[string]string{{"source": "quant-python", "error": message}})}, nil
}
