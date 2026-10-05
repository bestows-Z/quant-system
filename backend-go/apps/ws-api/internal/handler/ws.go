package handler

import (
	"net/http"
	"strings"
	"time"

	"quant-system/backend-go/apps/ws-api/internal/hub"
	"quant-system/backend-go/apps/ws-api/internal/svc"
	qjwt "quant-system/backend-go/common/jwt"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func WebSocketHandler(svcCtx *svc.ServiceContext, channel string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := tokenFromRequest(r)
		claims, err := qjwt.Parse(svcCtx.Config.JWT.Secret, token)
		if err != nil || claims.UserID == 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := hub.NewClient(claims.UserID, conn)
		client.Send(hub.Message{Type: "connected", Channel: channel, Data: map[string]any{"user_id": claims.UserID}, CreatedAt: time.Now()})
		svcCtx.Hub.Register(client)
		go client.WritePump()
		client.ReadPump(func(cmd hub.ClientCommand) {
			if cmd.Channel == "" {
				cmd.Channel = channel
			}
			var commandErr error
			switch cmd.Type {
			case "subscribe":
				commandErr = svcCtx.Hub.Subscribe(client, cmd)
			case "unsubscribe":
				svcCtx.Hub.Unsubscribe(client, cmd)
			default:
				commandErr = http.ErrNotSupported
			}
			if commandErr != nil {
				client.Send(hub.Message{Type: "command.error", Channel: cmd.Channel, Data: map[string]any{"command": cmd, "message": commandErr.Error()}, CreatedAt: time.Now()})
				return
			}
			client.Send(hub.Message{Type: cmd.Type + ".ack", Channel: cmd.Channel, Data: cmd, CreatedAt: time.Now()})
		})
		svcCtx.Hub.Unregister(client)
	}
}

func tokenFromRequest(r *http.Request) string {
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return auth
}
