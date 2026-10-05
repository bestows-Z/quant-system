package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"quant-system/backend-go/apps/agent-rpc/agentclient"
	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/gateway-api/internal/types"
)

func AnalyzeStockHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.AnalyzeStockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Agent.AnalyzeStock(r.Context(), &agentclient.AnalyzeStockReq{UserId: userID, Symbol: req.Symbol, Question: req.Question, KnowledgeBaseId: req.KnowledgeBaseID, AccountId: req.AccountID})
		writeAgentResult(w, r, reply, err)
	}
}

func AgentAskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.AgentAskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Agent.Ask(r.Context(), &agentclient.AgentAskReq{UserId: userID, Question: req.Question, Symbol: req.Symbol, KnowledgeBaseId: req.KnowledgeBaseID, AccountId: req.AccountID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Result)
	}
}

func ExplainSignalHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.ExplainSignalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Agent.ExplainSignal(r.Context(), &agentclient.ExplainSignalReq{UserId: userID, Symbol: req.Symbol, Question: req.Question, KnowledgeBaseId: req.KnowledgeBaseID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Result)
	}
}

func ReviewBacktestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.ReviewBacktestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Agent.ReviewBacktest(r.Context(), &agentclient.ReviewBacktestReq{UserId: userID, BacktestId: req.BacktestID, Question: req.Question, KnowledgeBaseId: req.KnowledgeBaseID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Result)
	}
}

func ReviewPaperAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.ReviewPaperAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Agent.ReviewPaperAccount(r.Context(), &agentclient.ReviewPaperAccountReq{UserId: userID, AccountId: req.AccountID, Question: req.Question, KnowledgeBaseId: req.KnowledgeBaseID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Result)
	}
}

func ListAgentTasksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
		reply, err := svcCtx.Agent.ListTasks(r.Context(), &agentclient.ListAgentTasksReq{UserId: userID, Status: r.URL.Query().Get("status"), TaskType: r.URL.Query().Get("task_type"), Limit: limit})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func GetAgentTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Agent.GetTask(r.Context(), &agentclient.GetAgentTaskReq{UserId: userID, TaskId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func GetAgentReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Agent.GetReport(r.Context(), &agentclient.GetAgentReportReq{UserId: userID, ReportId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func ListAgentReportsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
		offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
		reply, err := svcCtx.Agent.ListReports(r.Context(), &agentclient.ListAgentReportsReq{UserId: userID, Symbol: r.URL.Query().Get("symbol"), ReportType: r.URL.Query().Get("report_type"), StartDate: r.URL.Query().Get("start_date"), EndDate: r.URL.Query().Get("end_date"), Limit: limit, Offset: offset})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"items": reply.Items, "total": reply.Total})
	}
}

func DeleteAgentReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Agent.DeleteReports(r.Context(), &agentclient.DeleteAgentReportsReq{UserId: userID, ReportIds: []uint64{id}})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"deleted_count": reply.GetDeletedCount()})
	}
}

func DeleteAgentReportsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req struct {
			IDs []uint64 `json:"ids"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || len(req.IDs) == 0 {
			writeError(w, r, 10001, "请选择要删除的分析档案")
			return
		}
		reply, err := svcCtx.Agent.DeleteReports(r.Context(), &agentclient.DeleteAgentReportsReq{UserId: userID, ReportIds: req.IDs})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"deleted_count": reply.GetDeletedCount()})
	}
}

func CreateAgentChatSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req struct {
			Title string `json:"title"`
		}
		if r.ContentLength > 0 && json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Agent.CreateChatSession(r.Context(), &agentclient.CreateChatSessionReq{UserId: userID, Title: req.Title})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}
func ListAgentChatSessionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Agent.ListChatSessions(r.Context(), &agentclient.ListChatSessionsReq{UserId: userID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}
func RenameAgentChatSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		var req struct {
			Title string `json:"title"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Agent.RenameChatSession(r.Context(), &agentclient.RenameChatSessionReq{UserId: userID, SessionId: id, Title: req.Title})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}
func DeleteAgentChatSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Agent.DeleteChatSession(r.Context(), &agentclient.ChatSessionReq{UserId: userID, SessionId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"affected": reply.Affected})
	}
}
func ClearAgentChatMessagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Agent.ClearChatMessages(r.Context(), &agentclient.ChatSessionReq{UserId: userID, SessionId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"affected": reply.Affected})
	}
}
func ListAgentChatMessagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Agent.ListChatMessages(r.Context(), &agentclient.ChatSessionReq{UserId: userID, SessionId: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func writeAgentResult(w http.ResponseWriter, r *http.Request, reply *agentclient.AnalyzeStockResp, err error) {
	if err != nil {
		writeRPCError(w, r, err)
		return
	}
	writeSuccess(w, r, reply)
}
