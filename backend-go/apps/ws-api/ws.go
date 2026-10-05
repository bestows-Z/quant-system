package main

import (
	"flag"
	"fmt"

	"quant-system/backend-go/apps/ws-api/internal/config"
	"quant-system/backend-go/apps/ws-api/internal/handler"
	"quant-system/backend-go/apps/ws-api/internal/svc"
	"quant-system/backend-go/common/middleware"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/ws-api.yaml", "the config file")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	svcCtx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()
	server.Use(rest.ToMiddleware(middleware.Recover))
	server.Use(rest.ToMiddleware(middleware.RequestID))
	server.Use(rest.ToMiddleware(middleware.AccessLog))
	for _, ch := range []string{"market", "watchlist", "signal", "paper", "risk", "agent", "notify"} {
		channel := ch
		server.AddRoute(rest.Route{Method: "GET", Path: "/ws/" + channel, Handler: handler.WebSocketHandler(svcCtx, channel)})
	}
	server.AddRoute(rest.Route{Method: "GET", Path: "/ws/agent-chat", Handler: handler.AgentChatWebSocketHandler(svcCtx)})
	fmt.Printf("Starting ws api server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
