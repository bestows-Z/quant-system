package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	MySQL       MySQLConf
	CacheRedis  RedisConf
	RabbitMQ    RabbitMQConf
	QuantPython QuantPythonConf
	MarketCache MarketCacheConf
	AppLog      LogConf
}

type MySQLConf struct{ DSN string }
type RedisConf struct{ Host, Type, Pass string }
type RabbitMQConf struct{ URL, Exchange string }
type QuantPythonConf struct {
	BaseURL string
	Timeout time.Duration
}
type MarketCacheConf struct {
	RealtimeTTL          time.Duration
	StockListTTL         time.Duration
	KlineTTL             time.Duration
	AllowStaleCache      bool
	StaleCacheMaxSeconds int
}
type LogConf struct{ Level string }
