package main

import (
	"log"

	"github.com/labstack/echo/v4"
)

func main() {
	db, err := initDB()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	app := &App{DB: db}
	e := echo.New()
	msgs := e.Group("/messages")
	msgs.GET("", app.GetHandler)
	msgs.POST("", app.PostHandler)
	msgs.PATCH("/:id", app.PatchHandler)
	msgs.DELETE("/:id", app.DeleteHandler)

	e.Logger.Fatal(e.Start(":8080"))

}
