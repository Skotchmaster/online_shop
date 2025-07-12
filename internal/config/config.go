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
	JWT_SECRET     string
	REFRESH_SECRET string
	KAFKA_ADDRESS  string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}

	config := &Config{
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
