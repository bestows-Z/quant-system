package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL      MySQLConf
	Encryption EncryptionConf
}
type MySQLConf struct{ DSN string }
type EncryptionConf struct{ Key string }
