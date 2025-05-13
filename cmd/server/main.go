package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/Skotchmaster/online_shop/internal/jwtmiddleware"
)

func main() {
	db, err := handlers.InitDB()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	accessSecret := []byte(os.Getenv("ACCESS_SECRET"))
	refreshSecret := []byte(os.Getenv("REFRESH_SECRET"))

	productHandler := &handlers.ProductHandler{DB: db}
	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: accessSecret, RefreshSecret: refreshSecret}
	cartHandler := &handlers.CartHandler{DB: db}
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/register", authHandler.Register)
	e.POST("/login", authHandler.Login)

	api := e.Group("", jwtmiddleware.JWTMiddleware(accessSecret))

	api.POST("/product", productHandler.CreateProduct)
	api.PATCH("/product/:id", productHandler.PatchProduct)
	api.DELETE("/product/:id", productHandler.DeleteProduct)

	api.GET("/cart", cartHandler.GetCart)
	api.POST("/cart", cartHandler.AddToCart)
	api.DELETE("/cart/:id", cartHandler.DeleteOneFromCart)
	api.DELETE("/cart/:id", cartHandler.DeleteAllFromCart)

	e.Logger.Fatal(e.Start(":8080"))

}
