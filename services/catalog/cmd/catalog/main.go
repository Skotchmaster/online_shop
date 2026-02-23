package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/Skotchmaster/online_shop/pkg/authclient"
	pkgdb "github.com/Skotchmaster/online_shop/pkg/db"
	"github.com/Skotchmaster/online_shop/pkg/logging"
	loggingmw "github.com/Skotchmaster/online_shop/pkg/middleware/logging"

	catalogcfg "github.com/Skotchmaster/online_shop/services/catalog/internal/config"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/httpserver"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/repo"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/service"
)

func main() {
	if err := godotenv.Load("services/catalog/.env"); err != nil {
		log.Printf("warning: could not load .env: %v", err)
	}

	cfg := catalogcfg.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := pkgdb.Open(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatalf("db open: %v", err)
	}

	logger := logging.New(os.Getenv("LOG_LEVEL")).With("service", cfg.ServiceName)
	slog.SetDefault(logger)

	repo := &repo.GormRepo{DB: db}
	svc := &service.CatalogService{Repo: repo}
	handler := &httpserver.CatalogHTTP{Svc: svc}

	e := echo.New()
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	e.Use(loggingmw.RequestLogger(logger))
	e.Use(echomw.CORS())

	authclient := authclient.NewClient(cfg.AuthHTTPURL)

	httpserver.Register(e, &httpserver.Deps{
		CatalogHandler: handler,
		JWTSecret:      cfg.JWTAccessSecret,
		AuthClient:     authclient,
	})

	srv := &http.Server{
		Addr:              ":" + os.Getenv("SERVER_PORT"),
		Handler:           e,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
	}

	if srv.Addr == ":" {
		srv.Addr = ":8080"
	}

	go func() {
		log.Printf("catalog listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	_ = srv.Shutdown(shutdownCtx)

	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	log.Println("catalog stopped")
}