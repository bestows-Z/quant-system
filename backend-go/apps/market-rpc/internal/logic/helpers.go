package logic

import (
	"context"
	"strings"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/repository"
	"quant-system/backend-go/apps/market-rpc/internal/svc"
	"quant-system/backend-go/apps/market-rpc/market"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"
	"quant-system/backend-go/common/xmq"
)

func raw(jsonText string, err error) (*market.RawReply, error) {
	if err != nil {
		return nil, err
	}
	return &market.RawReply{Json: jsonText}, nil
}

func cacheAndPublishQuote(ctx context.Context, svcCtx *svc.ServiceContext, cacheKey string, rawJSON string, userID uint64) {
	if rawJSON == "" {
		return
	}
	if cacheKey != "" && svcCtx.Redis != nil {
		_ = svcCtx.Redis.Set(ctx, cacheKey, rawJSON, 30*time.Second).Err()
	}
	if svcCtx.Producer != nil {
		quotes := parseQuotes(rawData(rawJSON))
		for _, quote := range quotes {
			_ = svcCtx.Producer.Publish(ctx, xmq.Event{
				EventID:   idgen.NewEventID(),
				EventType: constants.EventMarketQuoteUpdated,
				TraceID:   trace.FromContext(ctx),
				UserID:    userID,
				Payload:   quoteMap(quote, valueOr(quote.Source, "realtime")),
				CreatedAt: time.Now(),
			})
		}
	}
}

func watchlistItem(item *repository.WatchlistItem) *market.WatchlistItem {
	if item == nil {
		return &market.WatchlistItem{}
	}
	remark := ""
	if item.Remark.Valid {
		remark = item.Remark.String
	}
	return &market.WatchlistItem{
		Id:        item.ID,
		UserId:    item.UserID,
		Symbol:    item.Symbol,
		GroupName: item.GroupName,
		Remark:    remark,
		CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func normalizeGroup(groupName string) string {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return "default"
	}
	return groupName
}
