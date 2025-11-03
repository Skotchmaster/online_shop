package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Skotchmaster/online_shop/pkg/middleware/auth"
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

	GormRepo := repo.GormRepo{
		DB: db,
		JWTSecret: cfg.JWTSecret,
		RefreshSecret: cfg.RefreshSecret,
	}

	AuthHTTP := httpserver.AuthHTTP{
		Svc: &service.AuthService{
			Repo: GormRepo,
		},
	}

	httpserver.Register(e, &httpserver.Deps{
		AuthHandler: AuthHTTP,
		Token: auth.TokenService{
			DB: db,
			JWTSecret: cfg.JWTSecret,
			RefreshSecret: cfg.RefreshSecret,
		},
	})

	go func() {
		if err := e.Start(cfg.AuthURL); err != nil && err != http.ErrServerClosed {
			log.Fatalf("echo start: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("echo shutdown: %v", err)
	}
}