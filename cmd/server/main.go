package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Skotchmaster/online_shop/internal/config"
	"github.com/Skotchmaster/online_shop/internal/es"
	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/Skotchmaster/online_shop/internal/handlers/cart"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/Skotchmaster/online_shop/internal/service/token"
	httpserver "github.com/Skotchmaster/online_shop/internal/transport/http"
)

func main() {
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	configuration, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	jwtSecret := []byte(configuration.JWT_SECRET)
	refreshSecret := []byte(configuration.REFRESH_SECRET)

	brokers := []string{configuration.KAFKA_ADDRESS}
	topics := []string{"user_events", "cart_events", "product_events"}
	prod, err := mykafka.NewProducer(brokers, topics)
	if err != nil {
		log.Fatal(err)
	}

	esClient, err := es.NewClient(configuration)
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover(), middleware.RequestID())

	deps := httpserver.Deps{
		DB: db,
		AuthHandler: &handlers.AuthHandler{DB: db, JWTSecret: jwtSecret, RefreshSecret: refreshSecret, Producer: prod},
		ProductHandler: &handlers.ProductHandler{DB: db, Producer: prod, JWTSecret: jwtSecret},
		CartHandler: &cart.CartHandler{DB: db, Producer: prod, JWTSecret: jwtSecret},
		ServiceHandler: &token.TokenService{DB: db, RefreshSecret: refreshSecret, JWTSecret: jwtSecret},
		SearchHandler: &handlers.SearchHandler{ES: esClient, Index: "product"},
	}

	httpserver.Register(e, &deps)

	srv := &http.Server{
	Addr:         ":8080",
	Handler:      e,
	ReadTimeout:  10 * time.Second,
	WriteTimeout: 15 * time.Second,
	IdleTimeout:  60 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("force exit")
		os.Exit(1)
	}()

	<-quit

	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("db close error: %v", err)
		}
	} else {
		log.Printf("db() error: %v", err)
	}

	if err := prod.Close(); err != nil {
		log.Printf("kafka close error: %v", err)
	}

	log.Println("shutdown complete")

}