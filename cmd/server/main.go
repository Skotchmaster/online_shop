package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/Skotchmaster/online_shop/internal/jwtmiddleware"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
)

const (
	topic = "my-topic"
)

var address = []string{"kafka:9092"}

func main() {
	db, err := handlers.InitDB()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	accessV1 := []byte(os.Getenv("ACCESS_SECRET_V1"))
	refreshV1 := []byte(os.Getenv("REFRESH_SECRET_V2"))

	accessKeyStore := map[string][]byte{
		"v1": accessV1,
	}

	p, err := mykafka.NewProducer(address)
	if err != nil {
		log.Fatalf("Kafka producer init error: %v", err)
	}
	defer p.Close()

	productHandler := &handlers.ProductHandler{DB: db, Producer: p}
	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: accessV1, RefreshSecret: refreshV1, Producer: p}
	cartHandler := &handlers.CartHandler{DB: db, Producer: p}
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/register", authHandler.Register)
	e.POST("/login", authHandler.Login)

	api := e.Group("", jwtmiddleware.JWTMiddleware(accessKeyStore))

	api.POST("/product", productHandler.CreateProduct)
	api.PATCH("/product/:id", productHandler.PatchProduct)
	api.DELETE("/product/:id", productHandler.DeleteProduct)

	api.GET("/cart", cartHandler.GetCart)
	api.POST("/cart", cartHandler.AddToCart)
	api.DELETE("/cart/:id", cartHandler.DeleteOneFromCart)
	api.DELETE("/cart/:id", cartHandler.DeleteAllFromCart)

	e.Logger.Fatal(e.Start(":8080"))

}
