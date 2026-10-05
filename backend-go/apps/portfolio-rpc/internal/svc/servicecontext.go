package svc

import (
	"database/sql"
	"os"

	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/portfolio-rpc/internal/config"
	"quant-system/backend-go/apps/portfolio-rpc/internal/domain"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config  config.Config
	DB      *sql.DB
	Service *domain.Service
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", os.ExpandEnv(c.MySQL.DSN))
	if err != nil {
		logx.Must(err)
	}
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	market := marketclient.NewMarket(zrpc.MustNewClient(c.MarketRpc))
	return &ServiceContext{Config: c, DB: db, Service: domain.NewService(db, market)}
}
