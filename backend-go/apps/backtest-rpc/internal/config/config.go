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
}

type MySQLConf struct{ DSN string }
type RedisConf struct{ Host, Type, Pass string }
type RabbitMQConf struct{ URL, Exchange string }
type QuantPythonConf struct {
	BaseURL string
	Timeout time.Duration
}
