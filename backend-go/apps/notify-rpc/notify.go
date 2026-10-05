package main

import (
	"flag"
	"fmt"

	"quant-system/backend-go/apps/notify-rpc/internal/config"
	"quant-system/backend-go/apps/notify-rpc/internal/server"
	"quant-system/backend-go/apps/notify-rpc/internal/svc"
	"quant-system/backend-go/apps/notify-rpc/notify"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/notify.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		notify.RegisterNotifyServer(grpcServer, server.NewNotifyServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
