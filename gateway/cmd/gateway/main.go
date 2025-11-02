package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Skotchmaster/online_shop/gateway/internal/config"
	"github.com/Skotchmaster/online_shop/gateway/internal/httpserver"
	"github.com/labstack/echo/v4"
)

func main() {
	cfg := config.Load()

	e := echo.New()
	e.Server.ReadTimeout = 10 * time.Second
	e.Server.WriteTimeout = 15 * time.Second
	e.Server.ReadHeaderTimeout = 3 * time.Second

	if err := httpserver.Register(e, &httpserver.Deps{
		AuthURL:    cfg.AuthURL,
		CatalogURL: cfg.CatalogURL,
		CartURL:    cfg.CartURL,
		OrderURL:   cfg.OrderURL,
		JWTSecret:  cfg.JWTSecret,
	}); err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := e.Start(cfg.ListenAddr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
