package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL       MySQLConf
	CacheRedis  RedisConf
	RabbitMQ    RabbitMQConf
	MarketRpc   zrpc.RpcClientConf
	PaperRpc    zrpc.RpcClientConf
	StrategyRpc zrpc.RpcClientConf
	BacktestRpc zrpc.RpcClientConf
	AppLog      LogConf
}

type MySQLConf struct{ DSN string }
type RedisConf struct{ Host, Type, Pass string }
type RabbitMQConf struct{ URL, Exchange string }
type LogConf struct{ Level string }
