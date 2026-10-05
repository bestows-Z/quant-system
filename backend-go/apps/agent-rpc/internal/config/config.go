package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	MySQL       MySQLConf
	CacheRedis  RedisConf
	LLM         LLMConf
	RabbitMQ    RabbitMQConf
	MarketRpc   zrpc.RpcClientConf
	PaperRpc    zrpc.RpcClientConf
	RagRpc      zrpc.RpcClientConf
	StrategyRpc zrpc.RpcClientConf
	BacktestRpc zrpc.RpcClientConf
	RiskRpc     zrpc.RpcClientConf
	ModelGatewayRpc zrpc.RpcClientConf
	AppLog      LogConf
}

type MySQLConf struct{ DSN string }
type RedisConf struct{ Host, Type, Pass string }
type LLMConf struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
	Timeout  time.Duration
}
type RabbitMQConf struct{ URL, Exchange string }
type LogConf struct{ Level string }
