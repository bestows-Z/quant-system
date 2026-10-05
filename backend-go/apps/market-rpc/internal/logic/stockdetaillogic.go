package logic

import (
	"context"
	"database/sql"
	"encoding/json"

	"quant-system/backend-go/apps/market-rpc/internal/repository"
	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type StockDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStockDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StockDetailLogic {
	return &StockDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StockDetailLogic) StockDetail(in *market.SymbolRequest) (*market.RawReply, error) {
	if row, err := l.svcCtx.Market.GetStock(l.ctx, in.GetSymbol()); err == nil {
		payload := stockPayload([]repository.StockRow{*row}, 1, 1, 1, "database")
		if items, ok := payload["items"].([]map[string]any); ok && len(items) > 0 {
			return &market.RawReply{Json: successEnvelope(items[0])}, nil
		}
	} else if err != sql.ErrNoRows {
		l.Logger.Errorf("query stock detail cache failed: %v", err)
	}
	jsonText, err := l.svcCtx.QuantPython.Stocks(l.ctx, in.GetSymbol(), 1, 1)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonText), &payload); err != nil {
		return &market.RawReply{Json: jsonText}, nil
	}
	data, _ := payload["data"].(map[string]any)
	items, _ := data["items"].([]any)
	if len(items) > 0 {
		body, _ := json.Marshal(map[string]any{"code": 0, "message": "success", "data": items[0]})
		return &market.RawReply{Json: string(body)}, nil
	}
	return &market.RawReply{Json: jsonText}, nil
}
