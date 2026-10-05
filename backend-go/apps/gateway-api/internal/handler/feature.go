package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"quant-system/backend-go/apps/feature-rpc/featureservice"
	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/market-rpc/marketclient"
)

func MarketOverviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.MarketBreadth(r.Context(), &marketclient.Empty{})
		writeRawOrError(w, r, reply, err)
	}
}

func MarketSectorsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		reply, err := svcCtx.Feature.Sectors(r.Context(), &featureservice.SectorListReq{Type: q.Get("type"), SortBy: q.Get("sort_by"), Order: q.Get("order")})
		writeFeatureRaw(w, r, reply, err)
	}
}

func SectorDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Feature.SectorDetail(r.Context(), &featureservice.SectorReq{Code: pathValue(r, "code")})
		writeFeatureRaw(w, r, reply, err)
	}
}

func SectorConstituentsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		reply, err := svcCtx.Feature.SectorConstituents(r.Context(), &featureservice.SectorReq{Code: pathValue(r, "code"), SortBy: q.Get("sort_by"), Order: q.Get("order"), Limit: int32Query(q.Get("limit"), 500)})
		writeFeatureRaw(w, r, reply, err)
	}
}

func SectorTimeshareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Feature.SectorTimeshare(r.Context(), &featureservice.SectorReq{Code: pathValue(r, "code")})
		writeFeatureRaw(w, r, reply, err)
	}
}

func MarketScreenerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		reply, err := svcCtx.Feature.Screener(r.Context(), &featureservice.ScreenerReq{
			SortBy: q.Get("sort_by"), Order: q.Get("order"), Industry: q.Get("industry"),
			MinChange: floatQuery(q.Get("min_change")), MaxChange: floatQuery(q.Get("max_change")),
			MinAmount: floatQuery(q.Get("min_amount")), Limit: int32Query(q.Get("limit"), 100),
		})
		writeFeatureRaw(w, r, reply, err)
	}
}

func CandidatesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		reply, err := svcCtx.Feature.Candidates(r.Context(), &featureservice.CandidateReq{
			SortBy: q.Get("sort_by"), Industry: q.Get("industry"), MinConfidence: floatQuery(q.Get("min_confidence")),
			Limit: int32Query(q.Get("limit"), 30), Refresh: q.Get("refresh") == "true",
		})
		writeFeatureRaw(w, r, reply, err)
	}
}

func CandidateEvidenceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Feature.CandidateEvidence(r.Context(), &featureservice.CandidateReq{Symbol: pathValue(r, "symbol"), Limit: 1})
		writeFeatureRaw(w, r, reply, err)
	}
}

func DragonTigerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		reply, err := svcCtx.Market.DragonTiger(r.Context(), &marketclient.ResearchRequest{TradeDate: q.Get("trade_date"), Limit: int32Query(q.Get("limit"), 50)})
		writeRawOrError(w, r, reply, err)
	}
}

func DragonTigerSeatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply, err := svcCtx.Market.DragonTigerSeats(r.Context(), &marketclient.ResearchRequest{
			Symbol: pathValue(r, "symbol"), TradeDate: r.URL.Query().Get("trade_date"),
		})
		writeRawOrError(w, r, reply, err)
	}
}

func writeFeatureRaw(w http.ResponseWriter, r *http.Request, reply *featureservice.RawReply, err error) {
	if err != nil {
		writeRPCError(w, r, err)
		return
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(reply.Json), &payload); err != nil {
		writeError(w, r, 10099, "市场分析响应解析失败")
		return
	}
	if code, ok := numericCode(payload["code"]); ok && code != 0 {
		message, _ := payload["message"].(string)
		writeErrorWithData(w, r, code, message, payload["data"])
		return
	}
	writeSuccess(w, r, payload["data"])
}

func floatQuery(value string) float64 {
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}
