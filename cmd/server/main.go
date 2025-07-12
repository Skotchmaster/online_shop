package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Skotchmaster/online_shop/internal/config"
	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/Skotchmaster/online_shop/internal/service"
)

func main() {
	db, err := handlers.InitDB()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	configuration, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	jwt_secret := []byte(configuration.JWT_SECRET)
	refresh := []byte(configuration.KAFKA_ADDRESS)

	brokers := []string{configuration.KAFKA_ADDRESS}
	topics := []string{"user_events", "cart_events", "product_events"}
	prod, err := mykafka.NewProducer(brokers, topics)
	if err != nil {
		log.Fatal(err)
	}
	defer prod.Close()

	productHandler := &handlers.ProductHandler{DB: db, Producer: prod, JWTSecret: jwt_secret}
	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: jwt_secret, RefreshSecret: refresh, Producer: prod}
	cartHandler := &handlers.CartHandler{DB: db, Producer: prod, JWTSecret: jwt_secret}
	serviceHandler := &service.TokenService{DB: db, RefreshSecret: refresh, JWTSecret: jwt_secret}
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/register", authHandler.Register)
	e.POST("/login", authHandler.Login)

	api := e.Group("/api")
	api.Use(serviceHandler.AutoRefreshMiddleware)

	api.GET("/product/:id", productHandler.GetProduct)
	api.POST("/product", productHandler.CreateProduct)
	api.PATCH("/product/:id", productHandler.PatchProduct)
	api.DELETE("/product/:id", productHandler.DeleteProduct)

	api.GET("/cart", cartHandler.GetCart)
	api.POST("/cart", cartHandler.AddToCart)
	api.DELETE("/cart/:id", cartHandler.DeleteOneFromCart)
	api.DELETE("/cart/:id", cartHandler.DeleteAllFromCart)

	e.Logger.Fatal(e.Start(":8080"))
}
