package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL      MySQLConf
	CacheRedis RedisConf
	Qdrant     QdrantConf
	LLM        LLMConf
	RabbitMQ   RabbitMQConf
	AppLog     LogConf
}

type MySQLConf struct{ DSN string }
type RedisConf struct{ Host, Type, Pass string }
type QdrantConf struct{ Addr, Collection string }
type LLMConf struct{ Provider, APIKey, BaseURL, Model string }
type RabbitMQConf struct{ URL, Exchange string }
type LogConf struct{ Level string }
