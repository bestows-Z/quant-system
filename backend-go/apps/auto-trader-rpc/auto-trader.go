package main

import (
	"flag"
	"fmt"

	"quant-system/backend-go/apps/auto-trader-rpc/autotrader"
	"quant-system/backend-go/apps/auto-trader-rpc/internal/config"
	"quant-system/backend-go/apps/auto-trader-rpc/internal/server"
	"quant-system/backend-go/apps/auto-trader-rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/auto-trader.yaml", "the config file")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)
	defer ctx.Close()
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		autotrader.RegisterAutoTraderServer(grpcServer, server.NewAutoTraderServer(ctx))
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()
	fmt.Printf("Starting auto trader rpc at %s...\n", c.ListenOn)
	s.Start()
}
