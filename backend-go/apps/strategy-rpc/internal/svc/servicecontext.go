package svc

import (
	"database/sql"
	"os"

	"quant-system/backend-go/apps/strategy-rpc/internal/client"
	"quant-system/backend-go/apps/strategy-rpc/internal/config"
	"quant-system/backend-go/apps/strategy-rpc/internal/domain"
	"quant-system/backend-go/common/xmq"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config   config.Config
	DB       *sql.DB
	Strategy *domain.Service
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", os.ExpandEnv(c.MySQL.DSN))
	if err != nil {
		logx.Must(err)
	}
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	quant := client.NewQuantPythonClient(c.QuantPython.BaseURL, c.QuantPython.Timeout)
	producer := xmq.NewRabbitProducer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange)
	return &ServiceContext{
		Config:   c,
		DB:       db,
		Strategy: domain.NewService(db, quant, producer),
	}
}
