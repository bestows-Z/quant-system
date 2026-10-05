package svc

import (
	"context"
	"database/sql"
	"os"

	"quant-system/backend-go/apps/auto-trader-rpc/internal/config"
	"quant-system/backend-go/apps/auto-trader-rpc/internal/domain"
	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/feature-rpc/featureservice"
	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgatewayservice"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/risk-rpc/riskclient"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config  config.Config
	Service *domain.Service
	db      *sql.DB
	cancel  context.CancelFunc
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", os.ExpandEnv(c.MySQL.DSN))
	if err != nil {
		logx.Must(err)
	}
	if err = db.Ping(); err != nil {
		logx.Must(err)
	}
	service := domain.NewService(db,
		featureservice.NewFeatureService(zrpc.MustNewClient(c.FeatureRpc)),
		strategyclient.NewStrategy(zrpc.MustNewClient(c.StrategyRpc)),
		riskclient.NewRisk(zrpc.MustNewClient(c.RiskRpc)),
		paperclient.NewPaper(zrpc.MustNewClient(c.PaperRpc)),
		marketclient.NewMarket(zrpc.MustNewClient(c.MarketRpc)),
		modelgatewayservice.NewModelGatewayService(zrpc.MustNewClient(c.ModelGatewayRpc)),
		backtestclient.NewBacktest(zrpc.MustNewClient(c.BacktestRpc)),
	)
	ctx, cancel := context.WithCancel(context.Background())
	go service.RunScheduler(ctx)
	return &ServiceContext{Config: c, Service: service, db: db, cancel: cancel}
}

func (s *ServiceContext) Close() {
	s.cancel()
	_ = s.db.Close()
}
