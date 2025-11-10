package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Skotchmaster/online_shop/services/auth/internal/config"
	"github.com/Skotchmaster/online_shop/services/auth/internal/httpserver"
	"github.com/Skotchmaster/online_shop/services/auth/internal/repo"
	"github.com/Skotchmaster/online_shop/services/auth/internal/service"
	"github.com/labstack/echo/v4"
)

func main() {
	cfg := config.Load()

	e := echo.New()
	e.Server.ReadTimeout = 10 * time.Second
	e.Server.WriteTimeout = 15 * time.Second
	e.Server.ReadHeaderTimeout = 3 * time.Second

	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := config.InitDB(initCtx)
	cancel()
	if err != nil {
		log.Fatalf("db init error: %v", err)
	}

	gormRepo := repo.GormRepo{
		DB:            db,
		JWTSecret:     cfg.JWTSecret,
		RefreshSecret: cfg.RefreshSecret,
	}

	authService := &service.AuthService{
		Repo: gormRepo,
	}

	authHandler := &httpserver.AuthHTTP{
		Svc: authService,
	}

	httpserver.Register(e, &httpserver.Deps{
		AuthHandler: authHandler,
	})

	// Запуск сервера
	go func() {
		if err := e.Start(cfg.AuthURL); err != nil && err != http.ErrServerClosed {
			log.Fatalf("echo start: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("echo shutdown: %v", err)
	}
	log.Println("Server stopped")
}