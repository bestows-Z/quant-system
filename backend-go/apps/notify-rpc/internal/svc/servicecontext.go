package svc

import (
	"context"
	"database/sql"
	"os"

	"quant-system/backend-go/apps/notify-rpc/internal/config"
	"quant-system/backend-go/apps/notify-rpc/internal/domain"
	"quant-system/backend-go/apps/notify-rpc/internal/repository"
	"quant-system/backend-go/common/xmq"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config config.Config
	Notify *domain.Service
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", os.ExpandEnv(c.MySQL.DSN))
	if err != nil {
		logx.Must(err)
	}
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	producer := xmq.NewRabbitProducer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange)
	notifySvc := domain.NewService(repository.NewNotificationRepository(db), repository.NewEventRepository(db), producer)
	consumer := xmq.NewRabbitConsumer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange, "quant.notify.events", []string{"strategy.signal.generated", "backtest.task.*", "paper.order.*", "risk.warning.created", "agent.task.*"})
	consumer.Start(context.Background(), notifySvc.HandleEvent)
	return &ServiceContext{
		Config: c,
		Notify: notifySvc,
	}
}
