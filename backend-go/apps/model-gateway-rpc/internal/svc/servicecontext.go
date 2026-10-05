package svc

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"quant-system/backend-go/apps/model-gateway-rpc/internal/config"
	"quant-system/backend-go/apps/model-gateway-rpc/internal/domain"
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
	if err = db.Ping(); err != nil {
		logx.Must(err)
	}
	crypto, err := domain.NewCipher(os.ExpandEnv(c.Encryption.Key))
	if err != nil {
		logx.Must(err)
	}
	return &ServiceContext{Config: c, DB: db, Service: domain.NewService(db, crypto)}
}
