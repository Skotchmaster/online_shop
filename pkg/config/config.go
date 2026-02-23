package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServiceName string

	ServerPort int

	DatabaseURL string

	JWTAccessSecret  []byte
	JWTRefreshSecret []byte

	AuthHTTPURL  string
	AuthGRPCAddr string

	KafkaBrokers []string
}

func Load() Config {
	return Config{
		ServiceName: EnvDefault("SERVICE_NAME", ""),

		ServerPort:  EnvIntDefault("SERVER_PORT", 8080),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		JWTAccessSecret:  []byte(os.Getenv("JWT_SECRET")),
		JWTRefreshSecret: []byte(os.Getenv("JWT_REFRESH_SECRET")),

		AuthHTTPURL:  os.Getenv("AUTH_URL"),
		AuthGRPCAddr: os.Getenv("AUTH_GRPC_ADDR"),

		KafkaBrokers: CSV(os.Getenv("KAFKA_BROKERS")),
	}
}

func CSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func EnvDefault(key, def string) string {
	if os.Getenv(key) != ""{
		return os.Getenv(key)
	}
	return def
}

func EnvIntDefault(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
