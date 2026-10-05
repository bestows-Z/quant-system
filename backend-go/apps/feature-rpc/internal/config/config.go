package config

import "github.com/zeromicro/go-zero/zrpc"

type MySQLConfig struct{ DSN string }

type Config struct {
	zrpc.RpcServerConf
	MySQL MySQLConfig
}
