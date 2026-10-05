package config

import (
	"time"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	JWT        JWTConf
	CacheRedis RedisConf
	RabbitMQ   RabbitMQConf
	NotifyRpc  zrpc.RpcClientConf
	MarketRpc  zrpc.RpcClientConf
	PaperRpc   zrpc.RpcClientConf
	AgentRpc   zrpc.RpcClientConf
	Stream     StreamConf
}

type JWTConf struct{ Secret string }
type RedisConf struct{ Host, Type, Pass string }
type RabbitMQConf struct{ URL, Exchange string }
type StreamConf struct {
	PollInterval      time.Duration
	TimeshareInterval time.Duration
	IdleInterval      time.Duration
	MaxConcurrency    int
}
