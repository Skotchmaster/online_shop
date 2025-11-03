package httpserver

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/middleware/auth"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	AuthHandler AuthHTTP
	Token auth.TokenService
}

func Register(e *echo.Echo, d *Deps) {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	api := e.Group("/api/v1")

	auth := api.Group("/auth")
	auth.POST("/register", d.AuthHandler.Register)
	auth.POST("/login", d.AuthHandler.Login)

	private := auth.Group("")
	private.Use(d.Token.AutoRefreshMiddleware)
	private.POST("/logout", d.AuthHandler.LogOut)

}