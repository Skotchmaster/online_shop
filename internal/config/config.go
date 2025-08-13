package config

import (
	"fmt"
	"log"
	"os"

	"github.com/Skotchmaster/online_shop/internal/models"
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
	ES_PORT        string
	ES_USER        string
	ES_PASSWORD    string
	ES_URL         string
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
		ES_PORT:        os.Getenv("ES_PORT"),
		ES_USER:        os.Getenv("ES_USER"),
		ES_PASSWORD:    os.Getenv("ES_PASSWORD"),
		ES_URL:         os.Getenv("ES_URL"),
		JWT_SECRET:     os.Getenv("JWT_SECRET"),
		REFRESH_SECRET: os.Getenv("REFRESH_SECRET"),
		KAFKA_ADDRESS:  os.Getenv("KAFKA_ADDRESS"),
	}

	return config, nil
}

func InitDB() (*gorm.DB, error) {
	configuration, _ := LoadConfig()

	host := configuration.DB_HOST
	port := configuration.DB_PORT
	user := configuration.DB_USER
	password := configuration.DB_PASSWORD
	dbname := configuration.DB_NAME

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}
	if err := db.AutoMigrate(&models.Product{}, &models.User{}, &models.RefreshToken{}, &models.CartItem{}, &models.Order{}, &models.OrderItem{}); err != nil {
		return nil, fmt.Errorf("не удалось выполнить миграцию: %w", err)
	}
	return db, nil
}
