package httpserver

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/authclient"
	middleware "github.com/Skotchmaster/online_shop/pkg/middleware/auth"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	CartHandler *CartHTTP
	JWTSecret  []byte
	AuthClient  *authclient.Client
}

func Register(e *echo.Echo, d *Deps) {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	authMW := middleware.NewAutoRefreshMiddleware(d.JWTSecret, d.AuthClient)

	cart := e.Group("/cart")
	cart.Use(authMW.RequireAuth)

	cart.GET("", d.CartHandler.GetCart)
	cart.POST("", d.CartHandler.AddToCart)
	cart.DELETE("", d.CartHandler.DeleteAllFromCart)
	cart.DELETE("/items", d.CartHandler.DeleteOneFromCart)
}