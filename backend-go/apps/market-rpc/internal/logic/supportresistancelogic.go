package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/repository"
	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type SupportResistanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSupportResistanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SupportResistanceLogic {
	return &SupportResistanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SupportResistanceLogic) SupportResistance(in *market.KlineRequest) (*market.RawReply, error) {
	jsonText, err := l.svcCtx.QuantPython.SupportResistance(l.ctx, in.GetSymbol(), in.GetStartDate(), in.GetEndDate())
	if err != nil {
		return nil, err
	}
	if item, ok := parseSupportResistance(jsonText); ok {
		if err := l.svcCtx.Market.SaveSupportResistance(l.ctx, item); err != nil {
			l.Errorf("save support resistance failed: %v", err)
		}
	}
	return &market.RawReply{Json: jsonText}, nil
}

func parseSupportResistance(raw string) (repository.SupportResistanceRow, bool) {
	var body struct {
		Code int `json:"code"`
		Data struct {
			Symbol             string  `json:"symbol"`
			Period             string  `json:"period"`
			QuoteTime          string  `json:"quote_time"`
			SupportPrice       float64 `json:"support_price"`
			PivotPrice         float64 `json:"pivot_price"`
			ResistancePrice    float64 `json:"resistance_price"`
			SupportStrength    float64 `json:"support_strength"`
			ResistanceStrength float64 `json:"resistance_strength"`
			Methods            any     `json:"methods"`
			AlgorithmVersion   string  `json:"algorithm_version"`
			DataSource         string  `json:"data_source"`
			ValidUntil         string  `json:"valid_until"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(raw), &body) != nil || body.Code != 0 || body.Data.Symbol == "" {
		return repository.SupportResistanceRow{}, false
	}
	quoteTime, err := parseAnalysisTime(body.Data.QuoteTime)
	if err != nil {
		return repository.SupportResistanceRow{}, false
	}
	validUntil, err := parseAnalysisTime(body.Data.ValidUntil)
	if err != nil {
		return repository.SupportResistanceRow{}, false
	}
	methods, _ := json.Marshal(body.Data.Methods)
	return repository.SupportResistanceRow{
		Symbol: body.Data.Symbol, Period: body.Data.Period, QuoteTime: quoteTime,
		SupportPrice: body.Data.SupportPrice, PivotPrice: body.Data.PivotPrice, ResistancePrice: body.Data.ResistancePrice,
		SupportStrength: body.Data.SupportStrength, ResistanceStrength: body.Data.ResistanceStrength,
		MethodJSON: string(methods), AlgorithmVersion: body.Data.AlgorithmVersion, DataSource: body.Data.DataSource, ValidUntil: validUntil,
	}, true
}

func parseAnalysisTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time: %s", value)
}
