package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL     MySQLConf
	MarketRpc zrpc.RpcClientConf
}

type MySQLConf struct{ DSN string }
