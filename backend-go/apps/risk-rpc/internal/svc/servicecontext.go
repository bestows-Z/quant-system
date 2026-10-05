package svc

import (
	"context"
	"database/sql"
	"os"

	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/risk-rpc/internal/config"
	"quant-system/backend-go/apps/risk-rpc/internal/domain"
	"quant-system/backend-go/apps/risk-rpc/internal/repository"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"
	"quant-system/backend-go/common/xmq"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	Risk   *domain.Service
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
	strategy := strategyclient.NewStrategy(zrpc.MustNewClient(c.StrategyRpc))
	backtest := backtestclient.NewBacktest(zrpc.MustNewClient(c.BacktestRpc))
	producer := xmq.NewRabbitProducer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange)
	riskSvc := domain.NewService(
		repository.NewRiskRecordRepository(db),
		repository.NewRiskRuleRepository(db),
		market,
		paper,
		strategy,
		backtest,
		producer,
	)
	riskSvc.EnsureRules(context.Background())
	return &ServiceContext{
		Config: c,
		Risk:   riskSvc,
	}
}
