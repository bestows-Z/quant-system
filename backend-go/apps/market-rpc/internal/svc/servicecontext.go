package svc

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"quant-system/backend-go/apps/market-rpc/internal/client"
	"quant-system/backend-go/apps/market-rpc/internal/config"
	"quant-system/backend-go/apps/market-rpc/internal/repository"
	"quant-system/backend-go/common/xmq"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config      config.Config
	DB          *sql.DB
	Redis       *redis.Client
	QuantPython *client.QuantPythonClient
	Market      *repository.MarketRepository
	Watchlists  *repository.WatchlistRepository
	Producer    xmq.Producer
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", os.ExpandEnv(c.MySQL.DSN))
	if err != nil {
		log.Fatalf("open mysql failed: %v", err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)

	host := c.CacheRedis.Host
	if !strings.Contains(host, ":") {
		host += ":6379"
	}
	marketRepo := repository.NewMarketRepository(db)
	marketRepo.EnsureSchema(context.Background())
	svcCtx := &ServiceContext{
		Config:      c,
		DB:          db,
		Redis:       redis.NewClient(&redis.Options{Addr: host, Password: c.CacheRedis.Pass}),
		QuantPython: client.NewQuantPythonClient(c.QuantPython.BaseURL, c.QuantPython.Timeout),
		Market:      marketRepo,
		Watchlists:  repository.NewWatchlistRepository(db),
		Producer:    xmq.NewRabbitProducer(c.RabbitMQ.URL, c.RabbitMQ.Exchange),
	}
	go svcCtx.runStockMasterSync()
	return svcCtx
}

func (s *ServiceContext) runStockMasterSync() {
	time.Sleep(2 * time.Second)
	s.syncStockMaster(false)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		s.syncStockMaster(true)
	}
}

func (s *ServiceContext) syncStockMaster(force bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	rows, total, err := s.Market.ListStocks(ctx, "", 1, 1)
	if err != nil {
		log.Printf("market stock prewarm skipped: database query failed: %v", err)
		return
	}
	const minimumStockMasterSize = 5000
	if !force && (total >= minimumStockMasterSize || len(rows) >= minimumStockMasterSize) {
		return
	}

	const pageSize int32 = 500
	items := make([]repository.StockRow, 0, pageSize)
	for page := int32(1); page <= 20; page++ {
		raw, err := s.QuantPython.Stocks(ctx, "", page, pageSize)
		if err != nil {
			log.Printf("market stock master sync failed: quant-python unavailable: %v", err)
			return
		}

		var body struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Items []struct {
					Symbol   string `json:"symbol"`
					Name     string `json:"name"`
					Market   string `json:"market"`
					Exchange string `json:"exchange"`
					Industry string `json:"industry"`
					Source   string `json:"source"`
				} `json:"items"`
				Total int `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			log.Printf("market stock master sync failed: invalid quant-python response: %v", err)
			return
		}
		if body.Code != 0 {
			log.Printf("market stock master sync skipped: source returned code=%d message=%s", body.Code, body.Message)
			return
		}
		if len(body.Data.Items) == 0 {
			break
		}

		for _, item := range body.Data.Items {
			if item.Symbol == "" {
				continue
			}
			items = append(items, repository.StockRow{
				Symbol:   item.Symbol,
				Name:     item.Name,
				Market:   item.Market,
				Exchange: item.Exchange,
				Industry: item.Industry,
				Source:   item.Source,
			})
		}
		if body.Data.Total > 0 && len(items) >= body.Data.Total {
			break
		}
		if len(body.Data.Items) < int(pageSize) {
			break
		}
	}
	if len(items) == 0 {
		log.Printf("market stock master sync skipped: no valid stock rows")
		return
	}
	if err := s.Market.UpsertStocks(ctx, items); err != nil {
		log.Printf("market stock master sync failed: database upsert failed: %v", err)
		return
	}
	log.Printf("market stock master sync completed: %d stocks stored", len(items))
}
