package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	MySQL           struct{ DSN string }
	FeatureRpc      zrpc.RpcClientConf
	StrategyRpc     zrpc.RpcClientConf
	RiskRpc         zrpc.RpcClientConf
	PaperRpc        zrpc.RpcClientConf
	MarketRpc       zrpc.RpcClientConf
	ModelGatewayRpc zrpc.RpcClientConf
	BacktestRpc     zrpc.RpcClientConf
}
