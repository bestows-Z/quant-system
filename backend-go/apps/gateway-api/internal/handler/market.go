package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/gateway-api/internal/types"
	"quant-system/backend-go/apps/market-rpc/marketclient"
	gwauth "quant-system/backend-go/common/auth"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func StocksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		reply, err := svcCtx.Market.Stocks(r.Context(), &marketclient.StockListRequest{
			Keyword:  query.Get("keyword"),
			Page:     int32Query(query.Get("page"), 1),
			PageSize: int32Query(query.Get("page_size"), 50),
		})
		writeRawOrError(w, r, reply, err)
	}
}

func SearchStocksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		reply, err := svcCtx.Market.SearchStocks(r.Context(), &marketclient.StockListRequest{
			Keyword:  query.Get("keyword"),
			Page:     int32Query(query.Get("page"), 1),
			PageSize: int32Query(query.Get("page_size"), 50),
		})
		writeRawOrError(w, r, reply, err)
	}
}

func StockDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.StockDetail(r.Context(), &marketclient.SymbolRequest{Symbol: pathValue(r, "symbol")})
		writeRawOrError(w, r, reply, err)
	}
}

func RealtimeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		reply, err := svcCtx.Market.Realtime(r.Context(), &marketclient.RealtimeRequest{
			Symbols: splitSymbols(query.Get("symbols")),
			Limit:   int32Query(query.Get("limit"), 50),
		})
		writeRawOrError(w, r, reply, err)
	}
}

func RealtimeBySymbolHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.RealtimeBySymbol(r.Context(), &marketclient.SymbolRequest{Symbol: pathValue(r, "symbol")})
		writeRawOrError(w, r, reply, err)
	}
}

func TimeshareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.Timeshare(r.Context(), &marketclient.SymbolRequest{Symbol: pathValue(r, "symbol")})
		writeRawOrError(w, r, reply, err)
	}
}

func OrderBookHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.OrderBook(r.Context(), &marketclient.SymbolRequest{Symbol: pathValue(r, "symbol")})
		writeRawOrError(w, r, reply, err)
	}
}

func KlineHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		reply, err := svcCtx.Market.Kline(r.Context(), &marketclient.KlineRequest{
			Symbol:    pathValue(r, "symbol"),
			Period:    query.Get("period"),
			StartDate: firstNonEmpty(query.Get("start_date"), query.Get("start")),
			EndDate:   firstNonEmpty(query.Get("end_date"), query.Get("end")),
			Adjust:    query.Get("adjust"),
		})
		writeRawOrError(w, r, reply, err)
	}
}

func IndicatorHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		reply, err := svcCtx.Market.Indicator(r.Context(), &marketclient.IndicatorRequest{
			Symbol:    pathValue(r, "symbol"),
			Type:      query.Get("type"),
			Period:    query.Get("period"),
			StartDate: firstNonEmpty(query.Get("start_date"), query.Get("start")),
			EndDate:   firstNonEmpty(query.Get("end_date"), query.Get("end")),
			Adjust:    query.Get("adjust"),
		})
		writeRawOrError(w, r, reply, err)
	}
}

func FullIndicatorHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		reply, err := svcCtx.Market.FullIndicator(r.Context(), &marketclient.IndicatorRequest{
			Symbol:    pathValue(r, "symbol"),
			Period:    query.Get("period"),
			StartDate: firstNonEmpty(query.Get("start_date"), query.Get("start")),
			EndDate:   firstNonEmpty(query.Get("end_date"), query.Get("end")),
			Adjust:    query.Get("adjust"),
		})
		writeRawOrError(w, r, reply, err)
	}
}

func SupportResistanceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		reply, err := svcCtx.Market.SupportResistance(r.Context(), &marketclient.KlineRequest{
			Symbol:    pathValue(r, "symbol"),
			Period:    "daily",
			StartDate: firstNonEmpty(query.Get("start_date"), query.Get("start")),
			EndDate:   firstNonEmpty(query.Get("end_date"), query.Get("end")),
			Adjust:    query.Get("adjust"),
		})
		writeRawOrError(w, r, reply, err)
	}
}

func MarketIndicesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.MarketIndices(r.Context(), &marketclient.Empty{})
		writeRawOrError(w, r, reply, err)
	}
}

func CompanyProfileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.CompanyProfile(r.Context(), &marketclient.SymbolRequest{Symbol: pathValue(r, "symbol")})
		writeRawOrError(w, r, reply, err)
	}
}

func AnnouncementsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.Announcements(r.Context(), researchRequest(r))
		writeRawOrError(w, r, reply, err)
	}
}

func NewsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := researchRequest(r)
		req.Keyword = r.URL.Query().Get("keyword")
		reply, err := svcCtx.Market.News(r.Context(), req)
		writeRawOrError(w, r, reply, err)
	}
}

func MoneyFlowHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.MoneyFlow(r.Context(), researchRequest(r))
		writeRawOrError(w, r, reply, err)
	}
}

func researchRequest(r *http.Request) *marketclient.ResearchRequest {
	return &marketclient.ResearchRequest{
		Symbol:    pathValue(r, "symbol"),
		TradeDate: r.URL.Query().Get("trade_date"),
		Limit:     int32Query(r.URL.Query().Get("limit"), 20),
	}
}

func WatchlistsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := gwauth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, r, 10002, "invalid token")
			return
		}
		reply, err := svcCtx.Market.Watchlists(r.Context(), &marketclient.WatchlistRequest{UserId: claims.UserID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func AddWatchlistHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := gwauth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, r, 10002, "invalid token")
			return
		}
		var req types.AddWatchlistRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Market.AddWatchlist(r.Context(), &marketclient.AddWatchlistRequest{
			UserId:    claims.UserID,
			Symbol:    req.Symbol,
			GroupName: req.GroupName,
			Remark:    req.Remark,
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func DeleteWatchlistHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := gwauth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, r, 10002, "invalid token")
			return
		}
		id, err := strconv.ParseUint(pathValue(r, "id"), 10, 64)
		if err != nil || id == 0 {
			writeError(w, r, 10001, "id 参数错误")
			return
		}
		_, err = svcCtx.Market.DeleteWatchlist(r.Context(), &marketclient.DeleteWatchlistRequest{UserId: claims.UserID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"deleted": true})
	}
}

func writeRawOrError(w http.ResponseWriter, r *http.Request, reply *marketclient.RawReply, err error) {
	if err != nil {
		writeRPCError(w, r, err)
		return
	}
	var payload any
	if err := json.Unmarshal([]byte(reply.Json), &payload); err != nil {
		writeError(w, r, 10099, "行情响应解析失败")
		return
	}
	if body, ok := payload.(map[string]any); ok {
		if code, ok := numericCode(body["code"]); ok && code != 0 {
			message, _ := body["message"].(string)
			if message == "" {
				message = "行情数据源不可用"
			}
			writeErrorWithData(w, r, code, message, body["data"])
			return
		}
	}
	writeSuccess(w, r, unwrapQuantPayload(payload))
}

func pathValue(r *http.Request, key string) string {
	if vars := pathvar.Vars(r); vars != nil {
		if value := vars[key]; value != "" {
			return value
		}
	}
	return r.URL.Query().Get(key)
}

func int32Query(value string, fallback int32) int32 {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return int32(parsed)
}

func splitSymbols(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	symbols := make([]string, 0, len(parts))
	for _, part := range parts {
		if symbol := strings.TrimSpace(part); symbol != "" {
			symbols = append(symbols, symbol)
		}
	}
	return symbols
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func numericCode(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

func unwrapQuantPayload(payload any) any {
	for i := 0; i < 3; i++ {
		body, ok := payload.(map[string]any)
		if !ok {
			return payload
		}
		if _, hasCode := body["code"]; !hasCode {
			return payload
		}
		data, hasData := body["data"]
		if !hasData {
			return payload
		}
		payload = data
	}
	return payload
}
