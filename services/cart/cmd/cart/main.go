package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Skotchmaster/online_shop/pkg/authclient"
	"github.com/Skotchmaster/online_shop/services/cart/internal/config"
	"github.com/Skotchmaster/online_shop/services/cart/internal/httpserver"
	"github.com/Skotchmaster/online_shop/services/cart/internal/repo"
	"github.com/Skotchmaster/online_shop/services/cart/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := config.Load()

	e := echo.New()
	e.HideBanner = true

	e.Server.ReadTimeout = 10 * time.Second
	e.Server.WriteTimeout = 15 * time.Second
	e.Server.ReadHeaderTimeout = 3 * time.Second

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := config.InitDB(initCtx)
	cancel()
	if err != nil {
		log.Fatalf("db init error: %v", err)
	}

	Repo := &repo.GormRepo{
		DB: db,
	}

	cartService := &service.CartService{
		Repo: Repo,
	}

	cartHandler := &httpserver.CartHTTP{
		Svc: cartService,
	}

	authClient := authclient.NewClient(cfg.AuthURL)

	httpserver.Register(e, &httpserver.Deps{
		CartHandler: cartHandler,
		JWTSecret:   cfg.JWTSecret,
		AuthClient:  authClient,
	})

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		log.Printf("Starting cart service on port %s...", port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
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

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	log.Println("Server stopped")
}
