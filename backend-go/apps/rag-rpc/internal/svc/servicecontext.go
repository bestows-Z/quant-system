package svc

import (
	"context"
	"database/sql"
	"os"
	"time"

	"quant-system/backend-go/apps/rag-rpc/internal/config"
	"quant-system/backend-go/apps/rag-rpc/internal/domain"
	"quant-system/backend-go/common/xmq"
	"quant-system/backend-go/common/xqdrant"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config config.Config
	DB     *sql.DB
	Rag    *domain.Service
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", os.ExpandEnv(c.MySQL.DSN))
	if err != nil {
		logx.Must(err)
	}
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	qdrant := xqdrant.New(c.Qdrant.Addr, c.Qdrant.Collection)
	producer := xmq.NewRabbitProducer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange)
	ragService := domain.NewService(db, qdrant, producer, c)
	if err := ragService.EnsureLifecycleSchema(context.Background()); err != nil {
		logx.Must(err)
	}
	ragService.StartVectorCleanupScheduler(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ragService.EnsureSystemKnowledge(ctx); err != nil {
		logx.Errorf("系统量化知识库初始化失败，将在服务重启时重试: %v", err)
	}
	return &ServiceContext{
		Config: c,
		DB:     db,
		Rag:    ragService,
	}
}
