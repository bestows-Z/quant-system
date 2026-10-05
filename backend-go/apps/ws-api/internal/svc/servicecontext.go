package svc

import (
	"context"
	"os"
	"strings"

	"quant-system/backend-go/apps/agent-rpc/agentclient"
	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/notify-rpc/notifyclient"
	"quant-system/backend-go/apps/paper-rpc/paperclient"
	"quant-system/backend-go/apps/ws-api/internal/config"
	"quant-system/backend-go/apps/ws-api/internal/hub"
	wsmq "quant-system/backend-go/apps/ws-api/internal/mq"
	"quant-system/backend-go/apps/ws-api/internal/stream"
	"quant-system/backend-go/common/xmq"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	Hub    *hub.Hub
	Notify notifyclient.Notify
	Market marketclient.Market
	Paper  paperclient.Paper
	Agent  agentclient.Agent
	Redis  *redis.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	h := hub.NewHub()
	host := c.CacheRedis.Host
	if !strings.Contains(host, ":") {
		host += ":6379"
	}
	market := marketclient.NewMarket(zrpc.MustNewClient(c.MarketRpc))
	paperTrading := paperclient.NewPaper(zrpc.MustNewClient(c.PaperRpc))
	ctx := &ServiceContext{
		Config: c, Hub: h, Notify: notifyclient.NewNotify(zrpc.MustNewClient(c.NotifyRpc)),
		Market: market, Paper: paperTrading, Agent: agentclient.NewAgent(zrpc.MustNewClient(c.AgentRpc)), Redis: redis.NewClient(&redis.Options{Addr: host, Password: c.CacheRedis.Pass}),
	}
	consumer := xmq.NewRabbitConsumer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange, "quant.ws.events", []string{"market.quote.updated", "strategy.signal.generated", "paper.order.filled", "risk.warning.created", "agent.task.*", "notify.created"})
	consumer.Start(context.Background(), wsmq.HandleEvent(h))
	stream.NewMarketStreamer(market, paperTrading, ctx.Redis, h, c.Stream).Start(context.Background())
	return ctx
}
