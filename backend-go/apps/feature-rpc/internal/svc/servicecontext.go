package svc

import (
	"context"
	"database/sql"
	"os"

	"quant-system/backend-go/apps/feature-rpc/internal/config"
	"quant-system/backend-go/apps/feature-rpc/internal/domain"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config  config.Config
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
	service := domain.NewService(db)
	if err = service.EnsureSectorSchema(context.Background()); err != nil {
		logx.Must(err)
	}
	service.StartCloseScheduler(context.Background())
	service.StartSectorSnapshotScheduler(context.Background())
	return &ServiceContext{Config: c, Service: service}
}
