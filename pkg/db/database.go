package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

func Open(ctx context.Context, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is empty")
	}

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
