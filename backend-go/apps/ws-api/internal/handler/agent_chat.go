package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"quant-system/backend-go/apps/agent-rpc/agentclient"
	"quant-system/backend-go/apps/ws-api/internal/hub"
	"quant-system/backend-go/apps/ws-api/internal/svc"
	"quant-system/backend-go/common/idgen"
	qjwt "quant-system/backend-go/common/jwt"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc/status"
)

func AgentChatWebSocketHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := qjwt.Parse(svcCtx.Config.JWT.Secret, tokenFromRequest(r))
		if err != nil || claims.UserID == 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadLimit(8192)
		_ = writeChatEvent(conn, "connected", "", map[string]any{"mode": "READ_ONLY", "disclaimer": "仅供研究，不构成投资建议"})
		for {
			var cmd hub.ClientCommand
			if err := conn.ReadJSON(&cmd); err != nil {
				return
			}
			traceID := idgen.NewEventID()
			if cmd.Type != "chat.ask" || cmd.SessionID == 0 || strings.TrimSpace(cmd.RequestID) == "" || strings.TrimSpace(cmd.Content) == "" {
				_ = writeChatEvent(conn, "chat.error", traceID, map[string]any{"code": "INVALID_REQUEST", "message": "请选择研究会话并输入问题"})
				continue
			}
			_ = writeChatEvent(conn, "chat.accepted", traceID, map[string]any{"request_id": cmd.RequestID, "session_id": cmd.SessionID})
			_ = writeChatEvent(conn, "chat.tool.started", traceID, map[string]any{"request_id": cmd.RequestID, "tool": "只读量化研究工具链"})
			ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
			reply, callErr := svcCtx.Agent.ChatAsk(ctx, &agentclient.AgentChatAskReq{UserId: claims.UserID, SessionId: cmd.SessionID, RequestId: cmd.RequestID, Content: cmd.Content, Symbol: cmd.Symbol, AccountId: cmd.AccountID, KnowledgeBaseId: cmd.KnowledgeBaseID})
			cancel()
			if callErr != nil {
				message := status.Convert(callErr).Message()
				if strings.TrimSpace(message) == "" {
					message = "智能研究服务暂时不可用，请稍后重试"
				}
				_ = writeChatEvent(conn, "chat.error", traceID, map[string]any{"code": "CHAT_FAILED", "message": message, "request_id": cmd.RequestID})
				continue
			}
			msg := reply.GetMessage()
			_ = writeChatEvent(conn, "chat.tool.completed", traceID, map[string]any{"request_id": cmd.RequestID, "tool": "只读量化研究工具链", "report_id": reply.GetReportId()})
			for _, part := range chatChunks(msg.GetContent(), 120) {
				if err := writeChatEvent(conn, "chat.delta", traceID, map[string]any{"request_id": cmd.RequestID, "content": part}); err != nil {
					return
				}
			}
			_ = writeChatEvent(conn, "chat.completed", traceID, map[string]any{"request_id": cmd.RequestID, "message": msg, "report_id": reply.GetReportId(), "citations_json": msg.GetCitationsJson(), "tool_calls_json": msg.GetToolCallsJson()})
		}
	}
}

func writeChatEvent(conn *websocket.Conn, eventType, traceID string, data any) error {
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(hub.Message{Type: eventType, Channel: "agent-chat", TraceID: traceID, Data: data, CreatedAt: time.Now()})
}

func chatChunks(text string, size int) []string {
	runes := []rune(text)
	if size <= 0 {
		size = 120
	}
	parts := make([]string, 0, (len(runes)+size-1)/size)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[start:end]))
	}
	return parts
}
