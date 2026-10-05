package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL      MySQLConf
	CacheRedis RedisConf
	JWT        JWTConf
	RabbitMQ   RabbitMQConf
	AppLog     LogConf
}

type MySQLConf struct{ DSN string }
type RedisConf struct{ Host, Type, Pass string }
type JWTConf struct {
	Secret      string
	ExpireHours int64
}
type RabbitMQConf struct{ URL, Exchange string }
type LogConf struct{ Level string }
