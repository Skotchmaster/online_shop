package config

import (
	"log"
	"os"
)

type Config struct {
	ListenAddr string
	AuthURL    string
	CatalogURL string
	CartURL    string
	OrderURL   string
	SearchURL  string
	JWT        string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func must(v string, name string) string {
	if v == "" {
		log.Fatalf("missing required env %s", name)
	}
	return v
}

func Load() *Config {
	cfg := &Config{
		ListenAddr: getenv("GATEWAY_ADDR", ":8080"),
		AuthURL:    must(os.Getenv("AUTH_URL"), "AUTH_URL"),
		CatalogURL: must(os.Getenv("CATALOG_URL"), "CATALOG_URL"),
		CartURL:    must(os.Getenv("CART_URL"), "CART_URL"),
		OrderURL:   must(os.Getenv("ORDER_URL"), "ORDER_URL"),
		SearchURL:  getenv("SEARCH_URL", ""),
		JWT:        must(os.Getenv("JWT_HS256_SECRET"), "JWT_HS256_SECRET"),
	}
	return cfg
}
