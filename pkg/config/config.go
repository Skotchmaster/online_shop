package config

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	DB_HOST        string
	DB_PORT        string
	DB_USER        string
	DB_PASSWORD    string
	DB_NAME        string
	JWT_SECRET     string
	REFRESH_SECRET string
	KAFKA_ADDRESS  string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Notice: .env file not found: %v. Using system environment variables", err)
	}

	config := &Config{
		DB_HOST:        os.Getenv("DB_HOST"),
		DB_PORT:        os.Getenv("DB_PORT"),
		DB_USER:        os.Getenv("DB_USER"),
		DB_PASSWORD:    os.Getenv("DB_PASSWORD"),
		DB_NAME:        os.Getenv("DB_NAME"),
		JWT_SECRET:     os.Getenv("JWT_SECRET"),
		REFRESH_SECRET: os.Getenv("REFRESH_SECRET"),
		KAFKA_ADDRESS:  os.Getenv("KAFKA_ADDRESS"),
	}

	return config, nil
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