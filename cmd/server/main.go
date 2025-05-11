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

	app := &handlers.ProductHandler{DB: db}
	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: accessSecret, RefreshSecret: refreshSecret}
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/register", authHandler.Register)
	e.POST("/login", authHandler.Login)

	api := e.Group("/product", jwtmiddleware.JWTMiddleware(accessSecret))

	api.POST("", app.CreateProduct)
	api.PATCH("", app.PatchProduct)

	e.Logger.Fatal(e.Start(":8080"))

}
