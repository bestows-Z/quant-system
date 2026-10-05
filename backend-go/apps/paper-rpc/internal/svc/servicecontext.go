package svc

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"quant-system/backend-go/apps/market-rpc/marketclient"
	"quant-system/backend-go/apps/paper-rpc/internal/config"
	"quant-system/backend-go/apps/paper-rpc/internal/domain"
	"quant-system/backend-go/common/xmq"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config  config.Config
	DB      *sql.DB
	Trading *domain.TradingService
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", os.ExpandEnv(c.MySQL.DSN))
	if err != nil {
		logx.Must(err)
	}
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	if err := ensurePaperSchema(context.Background(), db); err != nil {
		logx.Must(err)
	}
	market := marketclient.NewMarket(zrpc.MustNewClient(c.MarketRpc))
	producer := xmq.NewRabbitProducer(os.ExpandEnv(c.RabbitMQ.URL), c.RabbitMQ.Exchange)
	ctx := &ServiceContext{
		Config:  c,
		DB:      db,
		Trading: domain.NewTradingService(db, market, producer, c),
	}
	go ctx.runOrderExpiry(context.Background())
	return ctx
}

func (s *ServiceContext) runOrderExpiry(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if count, err := s.Trading.ExpireOrders(ctx, 500); err != nil {
				logx.Errorf("expire paper orders failed: %v", err)
			} else if count > 0 {
				logx.Infof("expired %d paper orders", count)
			}
		}
	}
}

func ensurePaperSchema(ctx context.Context, db *sql.DB) error {
	columns := map[string]string{
		"today_buy_quantity": "ALTER TABLE paper_positions ADD COLUMN today_buy_quantity INT NOT NULL DEFAULT 0 AFTER frozen_quantity",
		"last_buy_date":      "ALTER TABLE paper_positions ADD COLUMN last_buy_date DATE DEFAULT NULL AFTER today_buy_quantity",
	}
	for column, ddl := range columns {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='paper_positions' AND COLUMN_NAME=?`, column).Scan(&count); err != nil {
			return fmt.Errorf("inspect paper_positions.%s: %w", column, err)
		}
		if count == 0 {
			if _, err := db.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("add paper_positions.%s: %w", column, err)
			}
		}
	}
	return nil
}
