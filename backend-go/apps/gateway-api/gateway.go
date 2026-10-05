package main

import (
	"flag"
	"fmt"

	"quant-system/backend-go/apps/gateway-api/internal/config"
	"quant-system/backend-go/apps/gateway-api/internal/handler"
	"quant-system/backend-go/apps/gateway-api/internal/svc"
	"quant-system/backend-go/common/middleware"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/gateway-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	server.Use(rest.ToMiddleware(middleware.Recover))
	server.Use(rest.ToMiddleware(middleware.RequestID))
	server.Use(rest.ToMiddleware(middleware.CORS))
	server.Use(rest.ToMiddleware(middleware.AccessLog))
	handler.RegisterRoutes(server, ctx)

	fmt.Printf("Starting gateway-api at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
