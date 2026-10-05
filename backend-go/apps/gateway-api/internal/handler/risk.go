package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/risk-rpc/riskclient"
)

func ListRiskRecordsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		q := r.URL.Query()
		accountID, _ := strconv.ParseUint(q.Get("account_id"), 10, 64)
		limit, _ := strconv.ParseInt(q.Get("limit"), 10, 64)
		reply, err := svcCtx.Risk.ListRiskRecords(r.Context(), &riskclient.ListRiskRecordsReq{UserId: userID, AccountId: accountID, Symbol: q.Get("symbol"), RiskType: q.Get("risk_type"), RiskLevel: q.Get("risk_level"), Status: q.Get("status"), Limit: limit})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func GetRiskRecordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Risk.GetRiskRecord(r.Context(), &riskclient.GetRiskRecordReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func AnalyzeStockRiskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Risk.AnalyzeStockRisk(r.Context(), &riskclient.AnalyzeStockRiskReq{UserId: userID, Symbol: pathValue(r, "symbol")})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func AnalyzeAccountRiskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, accountID, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Risk.AnalyzeAccountRisk(r.Context(), &riskclient.AnalyzeAccountRiskReq{UserId: userID, AccountId: accountID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func AnalyzePositionRiskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, accountID, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Risk.AnalyzePositionRisk(r.Context(), &riskclient.AnalyzePositionRiskReq{UserId: userID, AccountId: accountID, Symbol: r.URL.Query().Get("symbol")})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func AnalyzeBacktestRiskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, taskID, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Risk.AnalyzeBacktestRisk(r.Context(), &riskclient.AnalyzeBacktestRiskReq{UserId: userID, TaskId: taskID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func ResolveRiskRecordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Risk.ResolveRiskRecord(r.Context(), &riskclient.ResolveRiskRecordReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func IgnoreRiskRecordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Risk.IgnoreRiskRecord(r.Context(), &riskclient.IgnoreRiskRecordReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func DeleteRiskRecordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Risk.DeleteRiskRecords(r.Context(), &riskclient.DeleteRiskRecordsReq{UserId: userID, Ids: []uint64{id}})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"deleted_count": reply.GetDeletedCount()})
	}
}

func DeleteRiskRecordsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req struct {
			IDs []uint64 `json:"ids"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || len(req.IDs) == 0 {
			writeError(w, r, 10001, "请选择要删除的风控记录")
			return
		}
		reply, err := svcCtx.Risk.DeleteRiskRecords(r.Context(), &riskclient.DeleteRiskRecordsReq{UserId: userID, Ids: req.IDs})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"deleted_count": reply.GetDeletedCount()})
	}
}
