package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL          MySQLConf
	CacheRedis     RedisConf
	RabbitMQ       RabbitMQConf
	MarketRpc      zrpc.RpcClientConf
	FeeConfig      FeeConfig
	SlippageConfig SlippageConfig
	MarketSafety   MarketSafetyConfig
	AppLog         LogConf
}

type MarketSafetyConfig struct{ MaxQuoteAgeSeconds int64 }

type MySQLConf struct{ DSN string }
type RedisConf struct{ Host, Type, Pass string }
type RabbitMQConf struct{ URL, Exchange string }
type FeeConfig struct {
	BuyFeeRate    string
	SellFeeRate   string
	MinFee        string
	StampDutyRate string
}
type SlippageConfig struct {
	MarketOrderSlippageRate string
}
type LogConf struct{ Level string }
