package logic

import (
	"context"

	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchStocksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchStocksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchStocksLogic {
	return &SearchStocksLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchStocksLogic) SearchStocks(in *market.StockListRequest) (*market.RawReply, error) {
	page, pageSize := int(in.GetPage()), int(in.GetPageSize())
	rows, total, err := l.svcCtx.Market.ListStocks(l.ctx, in.GetKeyword(), page, pageSize)
	if err == nil && len(rows) > 0 {
		return &market.RawReply{Json: successEnvelope(stockPayload(rows, total, page, pageSize, "database"))}, nil
	}
	jsonText, err := l.svcCtx.QuantPython.SearchStocks(l.ctx, in.GetKeyword(), in.GetPage(), in.GetPageSize())
	if err != nil {
		l.Errorf("online security search unavailable, serving database result: %v", err)
		return &market.RawReply{Json: successEnvelope(stockPayload(rows, total, page, pageSize, "database"))}, nil
	}
	if rawSuccess(jsonText) {
		items := parseStocks(rawData(jsonText))
		if err := l.svcCtx.Market.UpsertStocks(l.ctx, items); err != nil {
			l.Errorf("persist online security search result failed: %v", err)
		}
		return &market.RawReply{Json: jsonText}, nil
	}
	l.Errorf("online security search returned business error, serving database result")
	return &market.RawReply{Json: successEnvelope(stockPayload(rows, total, page, pageSize, "database"))}, nil
}
