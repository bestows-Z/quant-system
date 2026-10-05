package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/apps/gateway-api/internal/types"
	"quant-system/backend-go/apps/rag-rpc/ragclient"
)

func CreateKnowledgeBaseHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.CreateKnowledgeBaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Rag.CreateKnowledgeBase(r.Context(), &ragclient.CreateKnowledgeBaseReq{UserId: userID, Name: req.Name, Description: req.Description, Visibility: req.Visibility})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func ListKnowledgeBasesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Rag.ListKnowledgeBases(r.Context(), &ragclient.ListKnowledgeBasesReq{UserId: userID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func GetKnowledgeBaseHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Rag.GetKnowledgeBase(r.Context(), &ragclient.GetKnowledgeBaseReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func DeleteKnowledgeBaseHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Rag.DeleteKnowledgeBase(r.Context(), &ragclient.DeleteKnowledgeBaseReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"ok": reply.Ok, "vector_cleanup_pending": reply.VectorCleanupPending, "already_deleted": reply.AlreadyDeleted})
	}
}

func CreateRagDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.CreateDocumentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Rag.CreateDocument(r.Context(), &ragclient.CreateDocumentReq{
			UserId:          userID,
			KnowledgeBaseId: req.KnowledgeBaseID,
			Title:           req.Title,
			DocType:         req.DocType,
			SourceUrl:       req.SourceURL,
			Content:         req.Content,
			MetadataJson:    req.MetadataJSON,
		})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func ListRagDocumentsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		kbID, err := strconv.ParseUint(r.URL.Query().Get("knowledge_base_id"), 10, 64)
		if err != nil || kbID == 0 {
			writeError(w, r, 10001, "knowledge_base_id 参数错误")
			return
		}
		reply, err := svcCtx.Rag.ListDocuments(r.Context(), &ragclient.ListDocumentsReq{UserId: userID, KnowledgeBaseId: kbID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func GetRagDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Rag.GetDocument(r.Context(), &ragclient.GetDocumentReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Item)
	}
}

func DeleteRagDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, id, ok := userAndPathID(w, r)
		if !ok {
			return
		}
		reply, err := svcCtx.Rag.DeleteDocument(r.Context(), &ragclient.DeleteDocumentReq{UserId: userID, Id: id})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, map[string]any{"ok": reply.Ok, "vector_cleanup_pending": reply.VectorCleanupPending, "already_deleted": reply.AlreadyDeleted})
	}
}

func ListRagChunksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		docID, err := strconv.ParseUint(r.URL.Query().Get("document_id"), 10, 64)
		if err != nil || docID == 0 {
			writeError(w, r, 10001, "document_id 参数错误")
			return
		}
		reply, err := svcCtx.Rag.ListChunks(r.Context(), &ragclient.ListChunksReq{UserId: userID, DocumentId: docID})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply.Items)
	}
}

func RagSearchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.RagSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Rag.Search(r.Context(), &ragclient.SearchReq{UserId: userID, KnowledgeBaseId: req.KnowledgeBaseID, Query: req.Query, TopK: req.TopK})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func RagAskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(w, r)
		if !ok {
			return
		}
		var req types.RagAskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, 10001, "请求参数错误")
			return
		}
		reply, err := svcCtx.Rag.Ask(r.Context(), &ragclient.AskReq{UserId: userID, KnowledgeBaseId: req.KnowledgeBaseID, Question: req.Question, TopK: req.TopK})
		if err != nil {
			writeRPCError(w, r, err)
			return
		}
		writeSuccess(w, r, reply)
	}
}

func userAndPathID(w http.ResponseWriter, r *http.Request) (uint64, uint64, bool) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseUint(pathValue(r, "id"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, r, 10001, "id 参数错误")
		return 0, 0, false
	}
	return userID, id, true
}
