package config

import (
	"time"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	JWT         JWTConf
	MySQL       MySQLConf
	CacheRedis  RedisConf
	RabbitMQ    RabbitMQConf
	Qdrant      QdrantConf
	QuantPython QuantPythonConf
	LLM         LLMConf
	RateLimit   RateLimitConf
	BizTimeout  TimeoutConf

	UserRpc         zrpc.RpcClientConf
	MarketRpc       zrpc.RpcClientConf
	PaperRpc        zrpc.RpcClientConf
	RagRpc          zrpc.RpcClientConf
	AgentRpc        zrpc.RpcClientConf
	StrategyRpc     zrpc.RpcClientConf
	BacktestRpc     zrpc.RpcClientConf
	RiskRpc         zrpc.RpcClientConf
	TaskRpc         zrpc.RpcClientConf
	NotifyRpc       zrpc.RpcClientConf
	PortfolioRpc    zrpc.RpcClientConf
	ModelGatewayRpc zrpc.RpcClientConf
	FeatureRpc      zrpc.RpcClientConf
	AutoTraderRpc   zrpc.RpcClientConf
}

type JWTConf struct {
	Secret      string
	ExpireHours int64
}

type MySQLConf struct {
	DSN string
}

type RedisConf struct {
	Host string
	Type string
	Pass string
}

type RabbitMQConf struct {
	URL      string
	Exchange string
}

type QdrantConf struct {
	Addr       string
	Collection string
}

type QuantPythonConf struct {
	BaseURL string
	Timeout time.Duration
}

type LLMConf struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

type RateLimitConf struct {
	LoginPerMinute int
	AgentPerMinute int
	RagPerMinute   int
}

type TimeoutConf struct {
	External time.Duration
	LLM      time.Duration
}
