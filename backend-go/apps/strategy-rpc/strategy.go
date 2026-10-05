package main

import (
	"flag"
	"fmt"

	"quant-system/backend-go/apps/strategy-rpc/internal/config"
	"quant-system/backend-go/apps/strategy-rpc/internal/server"
	"quant-system/backend-go/apps/strategy-rpc/internal/svc"
	"quant-system/backend-go/apps/strategy-rpc/strategy"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/strategy.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		strategy.RegisterStrategyServer(grpcServer, server.NewStrategyServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
