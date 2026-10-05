package svc

import (
	"quant-system/backend-go/apps/agent-rpc/agentclient"
	"quant-system/backend-go/apps/auto-trader-rpc/autotraderservice"
	"quant-system/backend-go/apps/backtest-rpc/backtestclient"
	"quant-system/backend-go/apps/feature-rpc/featureservice"
	"quant-system/backend-go/apps/gateway-api/internal/config"
	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/model-gateway-rpc/modelgatewayservice"
	"quant-system/backend-go/apps/notify-rpc/notifyclient"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/portfolio-rpc/portfolioservice"
	"quant-system/backend-go/apps/rag-rpc/ragclient"
	"quant-system/backend-go/apps/risk-rpc/riskclient"
	"quant-system/backend-go/apps/strategy-rpc/strategyclient"
	"quant-system/backend-go/apps/task-rpc/taskclient"
	"quant-system/backend-go/apps/user-rpc/userclient"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	User         userclient.User
	Market       marketclient.Market
	Paper        paperclient.Paper
	Rag          ragclient.Rag
	Agent        agentclient.Agent
	Strategy     strategyclient.Strategy
	Backtest     backtestclient.Backtest
	Risk         riskclient.Risk
	Task         taskclient.Task
	Notify       notifyclient.Notify
	Portfolio    portfolioservice.PortfolioService
	ModelGateway modelgatewayservice.ModelGatewayService
	Feature      featureservice.FeatureService
	AutoTrader   autotraderservice.AutoTrader
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		User:         userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		Market:       marketclient.NewMarket(zrpc.MustNewClient(c.MarketRpc)),
		Paper:        paperclient.NewPaper(zrpc.MustNewClient(c.PaperRpc)),
		Rag:          ragclient.NewRag(zrpc.MustNewClient(c.RagRpc)),
		Agent:        agentclient.NewAgent(zrpc.MustNewClient(c.AgentRpc)),
		Strategy:     strategyclient.NewStrategy(zrpc.MustNewClient(c.StrategyRpc)),
		Backtest:     backtestclient.NewBacktest(zrpc.MustNewClient(c.BacktestRpc)),
		Risk:         riskclient.NewRisk(zrpc.MustNewClient(c.RiskRpc)),
		Task:         taskclient.NewTask(zrpc.MustNewClient(c.TaskRpc)),
		Notify:       notifyclient.NewNotify(zrpc.MustNewClient(c.NotifyRpc)),
		Portfolio:    portfolioservice.NewPortfolioService(zrpc.MustNewClient(c.PortfolioRpc)),
		ModelGateway: modelgatewayservice.NewModelGatewayService(zrpc.MustNewClient(c.ModelGatewayRpc)),
		Feature:      featureservice.NewFeatureService(zrpc.MustNewClient(c.FeatureRpc)),
		AutoTrader:   autotraderservice.NewAutoTrader(zrpc.MustNewClient(c.AutoTraderRpc)),
	}
}
