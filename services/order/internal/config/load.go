package config

import "github.com/Skotchmaster/online_shop/pkg/config"

type ServiceConfig struct {
	config.Config
}

func Load() ServiceConfig {
	cfg := config.Load()

	config.MustNonEmpty(cfg.DatabaseURL, "DATABASE_URL")
	config.MustNonEmptyBytes(cfg.JWTAccessSecret, "JWT_SECRET")
	config.MustNonEmpty(cfg.AuthHTTPURL, "AUTH_URL")

	return ServiceConfig{Config: cfg}
}
