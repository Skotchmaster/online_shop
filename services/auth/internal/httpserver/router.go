package httpserver

import (
	"net/http"
	"os"

	"github.com/Skotchmaster/online_shop/services/auth/internal/middleware"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	AuthHandler *AuthHTTP
}

func Register(e *echo.Echo, d *Deps) {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	authMw := middleware.NewSimpleAuth(jwtSecret)

	e.POST("/register", d.AuthHandler.Register)
	e.POST("/login", d.AuthHandler.Login)
	e.POST("/refresh", d.AuthHandler.Refresh)

	private := e.Group("")
	private.Use(authMw.RequireAuth)
	
	private.POST("/logout", d.AuthHandler.LogOut)
}