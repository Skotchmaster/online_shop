package httpserver

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/services/auth/internal/middleware"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	AuthHandler *AuthHTTP
	JWT_Secret []byte
}

func Register(e *echo.Echo, d *Deps) {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	authMw := middleware.NewSimpleAuth(d.JWT_Secret)

	e.POST("/register", d.AuthHandler.Register)
	e.POST("/login", d.AuthHandler.Login)
	e.POST("/refresh", d.AuthHandler.Refresh)

	private := e.Group("")
	private.Use(authMw.RequireAuth)
	
	private.POST("/logout", d.AuthHandler.LogOut)
}