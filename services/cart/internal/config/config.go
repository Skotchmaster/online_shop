package config

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	AuthURL       string
	JWTSecret     []byte
}

func must(v string, name string) string {
	if v == "" {
		log.Fatalf("missing required env %s", name)
	}
	return v
}

func Load() *Config {
	cfg := &Config{
		AuthURL:    must(os.Getenv("AUTH_URL"), "AUTH_URL"),
		JWTSecret:  []byte(must(os.Getenv("JWT_HS256_SECRET"), "JWT_HS256_SECRET")),
	}
	return cfg
}

func configurePool(sqlDB *sql.DB) {
	const (
		maxOpenConns    = 20
		maxIdleConns    = 10
		connMaxLifetime = 30 * time.Minute
		connMaxIdleTime = 5 * time.Minute
	)

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
}

func InitDB(ctx context.Context) (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		NowFunc:     func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("подключение к БД: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("получение sql.DB: %w", err)
	}
	configurePool(sqlDB)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping БД: %w", err)
	}

	return db, nil
}
