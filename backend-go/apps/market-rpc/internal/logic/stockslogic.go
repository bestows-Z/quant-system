package logic

import (
	"context"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type StocksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStocksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StocksLogic {
	return &StocksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StocksLogic) Stocks(in *market.StockListRequest) (*market.RawReply, error) {
	page, pageSize := int(in.GetPage()), int(in.GetPageSize())
	rows, total, err := l.svcCtx.Market.ListStocks(l.ctx, in.GetKeyword(), page, pageSize)
	if err == nil && len(rows) > 0 {
		return &market.RawReply{Json: successEnvelope(stockPayload(rows, total, page, pageSize, stockStorageSource(rows)))}, nil
	}
	jsonText, err := l.svcCtx.QuantPython.Stocks(l.ctx, in.GetKeyword(), in.GetPage(), in.GetPageSize())
	if err != nil {
		if len(rows) > 0 {
			return &market.RawReply{Json: successEnvelope(stockPayload(rows, total, page, pageSize, stockStorageSource(rows)))}, nil
		}
		return nil, err
	}
	if rawSuccess(jsonText) {
		data := rawData(jsonText)
		_ = l.svcCtx.Market.UpsertStocks(l.ctx, parseStocks(data))
		return &market.RawReply{Json: jsonText}, nil
	}
	if len(rows) > 0 {
		return &market.RawReply{Json: successEnvelope(stockPayload(rows, total, page, pageSize, stockStorageSource(rows)))}, nil
	}
	return &market.RawReply{Json: jsonText}, nil
}
