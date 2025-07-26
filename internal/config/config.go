package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
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
		return nil, fmt.Errorf("не удалось подключиться к env файлу: %w", err)
	}

	config := &Config{
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
