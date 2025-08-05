package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Skotchmaster/online_shop/internal/config"
	"github.com/Skotchmaster/online_shop/internal/es"
	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/Skotchmaster/online_shop/internal/handlers/cart"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/Skotchmaster/online_shop/internal/service/token"
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

	jwt_secret := []byte(configuration.JWT_SECRET)
	refresh := []byte(configuration.REFRESH_SECRET)

	brokers := []string{configuration.KAFKA_ADDRESS}
	topics := []string{"user_events", "cart_events", "product_events"}
	prod, err := mykafka.NewProducer(brokers, topics)
	if err != nil {
		log.Fatal(err)
	}
	defer prod.Close()

	es, err := es.NewClient(configuration)
	if err != nil {
		log.Fatal(err)
	}

	productHandler := &handlers.ProductHandler{DB: db, Producer: prod, JWTSecret: jwt_secret}
	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: jwt_secret, RefreshSecret: refresh, Producer: prod}
	cartHandler := &cart.CartHandler{DB: db, Producer: prod, JWTSecret: jwt_secret}
	serviceHandler := &token.TokenService{DB: db, RefreshSecret: refresh, JWTSecret: jwt_secret}
	searchHandler := &handlers.SearchHandler{ES: es, Index: "product"}
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/register", authHandler.Register)
	e.POST("/login", authHandler.Login)
	e.POST("/logout", authHandler.LogOut)

	e.GET("/search", searchHandler.Handler)
	api := e.Group("/api")
	api.Use(serviceHandler.AutoRefreshMiddleware)
	api_admin := e.Group("/admin")
	api_admin.Use(serviceHandler.AutoRefreshMiddlewareAdmin)

	api_admin.POST("/product", productHandler.CreateProduct)
	api_admin.PATCH("/product/:id", productHandler.PatchProduct)
	api_admin.DELETE("/product/:id", productHandler.DeleteProduct)

	api.GET("/product/:id", productHandler.GetProduct)
	api.GET("/cart", cartHandler.GetCart)
	api.POST("/cart", cartHandler.AddToCart)
	api.POST("/cart/order", cartHandler.MakeOrder)
	api.DELETE("/cart/:id", cartHandler.DeleteOneFromCart)
	api.DELETE("/cart/:id", cartHandler.DeleteAllFromCart)

	e.Logger.Fatal(e.Start(":8080"))
}
