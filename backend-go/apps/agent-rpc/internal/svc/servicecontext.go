package svc

import (
	"context"
	"database/sql"
	"os"

	"quant-system/backend-go/apps/agent-rpc/internal/config"
	"quant-system/backend-go/apps/agent-rpc/internal/domain"
	"quant-system/backend-go/apps/agent-rpc/internal/repository"
	"quant-system/backend-go/apps/agent-rpc/internal/tools"
	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgatewayservice"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/rag-rpc/ragclient"
	"quant-system/backend-go/apps/risk-rpc/riskclient"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"
	"quant-system/backend-go/common/xmq"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	DB           *sql.DB
	Orchestrator *domain.Orchestrator
	Chat         *domain.ChatService
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
	paper := paperclient.NewPaper(zrpc.MustNewClient(c.PaperRpc))
	rag := ragclient.NewRag(zrpc.MustNewClient(c.RagRpc))
	strategy := strategyclient.NewStrategy(zrpc.MustNewClient(c.StrategyRpc))
	backtest := backtestclient.NewBacktest(zrpc.MustNewClient(c.BacktestRpc))
	risk := riskclient.NewRisk(zrpc.MustNewClient(c.RiskRpc))
	producer := xmq.NewRabbitProducer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange)
	toolset := []domain.AgentTool{
		tools.MarketTool{Market: market},
		tools.IndicatorTool{Market: market},
		tools.StrategyTool{Strategy: strategy},
		tools.RiskTool{Risk: risk},
		tools.BacktestTool{Backtest: backtest},
		tools.RagSearchTool{Rag: rag},
		tools.PaperAccountTool{Paper: paper},
	}
	modelGateway := modelgatewayservice.NewModelGatewayService(zrpc.MustNewClient(c.ModelGatewayRpc))
	llm := domain.NewGatewayLLM(modelGateway)
	chatRepo := repository.NewChatRepository(db)
	orchestrator := domain.NewOrchestrator(
		repository.NewTaskRepository(db),
		repository.NewMessageRepository(db),
		repository.NewToolCallRepository(db),
		repository.NewReportRepository(db),
		toolset,
		llm,
		producer,
	)
	domain.StartCandidateScheduler(context.Background(), db, orchestrator)
	return &ServiceContext{
		Config:       c,
		DB:           db,
		Orchestrator: orchestrator,
		Chat:         domain.NewChatService(chatRepo, orchestrator),
	}
}
