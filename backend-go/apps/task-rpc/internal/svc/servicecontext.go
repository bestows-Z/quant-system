package svc

import (
	"context"
	"database/sql"
	"os"

	"quant-system/backend-go/apps/task-rpc/internal/config"
	"quant-system/backend-go/apps/task-rpc/internal/domain"
	"quant-system/backend-go/apps/task-rpc/internal/repository"
	"quant-system/backend-go/common/xmq"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config config.Config
	Task   *domain.Service
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", os.ExpandEnv(c.MySQL.DSN))
	if err != nil {
		logx.Must(err)
	}
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	taskSvc := domain.NewService(repository.NewTaskRepository(db), repository.NewTaskLogRepository(db))
	consumer := xmq.NewRabbitConsumer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange, "quant.task.events", []string{"*.task.*", "backtest.task.*", "agent.task.*", "strategy.*", "risk.warning.created", "rag.*"})
	consumer.Start(context.Background(), taskSvc.HandleEvent)
	return &ServiceContext{
		Config: c,
		Task:   taskSvc,
	}
}
