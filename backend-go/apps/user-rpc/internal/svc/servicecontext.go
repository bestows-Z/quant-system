package svc

import (
	"database/sql"
	"log"
	"os"

	"quant-system/backend-go/apps/user-rpc/internal/config"
	"quant-system/backend-go/apps/user-rpc/internal/repository"

	_ "github.com/go-sql-driver/mysql"
)

type ServiceContext struct {
	Config config.Config
	DB     *sql.DB
	Users  *repository.UserRepository
}

func NewServiceContext(c config.Config) *ServiceContext {
	dsn := os.ExpandEnv(c.MySQL.DSN)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open mysql failed: %v", err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	return &ServiceContext{
		Config: c,
		DB:     db,
		Users:  repository.NewUserRepository(db),
	}
}
