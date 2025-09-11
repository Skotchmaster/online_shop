package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Skotchmaster/online_shop/internal/config"
	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/Skotchmaster/online_shop/internal/handlers/cart"
	"github.com/Skotchmaster/online_shop/internal/logging"
	"github.com/Skotchmaster/online_shop/internal/middleware/auth"
	loggingmw "github.com/Skotchmaster/online_shop/internal/middleware/logging"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	httpserver "github.com/Skotchmaster/online_shop/internal/transport/http"
)

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}

	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := config.InitDB(initCtx)
	cancel()
	if err != nil {
		log.Fatalf("db init error: %v", err)
	}

	brokers := []string{cfg.KAFKA_ADDRESS}
	topics := []string{"user_events", "product_events", "cart_events"}
	producer, err := mykafka.NewProducer(brokers, topics)
	if err != nil {
		log.Fatalf("kafka init error: %v", err)
	}

	jwtSecretStr := cfg.JWT_SECRET
	if jwtSecretStr == "" {
		log.Printf("warning: JWT_SECRET is empty")
	}
	if cfg.REFRESH_SECRET == "" {
		log.Fatal("REFRESH secret is empty")
	}
	jwtSecret := []byte(jwtSecretStr)
	refreshSecret := []byte(cfg.REFRESH_SECRET)

	logger := logging.New(getenv("LOG_LEVEL", "info"))
	slog.SetDefault(logger)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover(), middleware.RequestID(), loggingmw.RequestLogger(logger), middleware.CORS())

	e.GET("/healthz", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	productHandler := &handlers.ProductHandler{DB: db, Producer: producer, JWTSecret: jwtSecret}
	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: jwtSecret, RefreshSecret: refreshSecret, Producer: producer}
	cartHandler := &cart.CartHandler{DB: db, Producer: producer, JWTSecret: jwtSecret}
	searchHandler := &handlers.SearchHandler{DB: db}
	tokenSvc := &auth.TokenService{DB: db, JWTSecret: jwtSecret, RefreshSecret: refreshSecret}

	httpserver.Register(e, &httpserver.Deps{
		DB:             db,
		ProductHandler: productHandler,
		AuthHandler:    authHandler,
		CartHandler:    cartHandler,
		ServiceHandler: tokenSvc,
		SearchHandler:  searchHandler,
	})

	port := getenv("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      e,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Printf("server is listening on :%s", port)
		if err := e.StartServer(srv); err != nil {
			srvErr <- err
		}
	}()

	select {
	case s := <-sigCh:
		log.Printf("got signal: %v", s)
	case err := <-srvErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}

	log.Println("shutting down...")
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("db close error: %v", err)
		}
	} else {
		log.Printf("db() error: %v", err)
	}

	if err := producer.Close(); err != nil {
		log.Printf("kafka close error: %v", err)
	}

	log.Println("shutdown complete")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
