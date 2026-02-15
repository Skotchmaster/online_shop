package httpserver

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/authclient"
	middleware "github.com/Skotchmaster/online_shop/pkg/middleware/auth"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	OrderHandler *OrderHTTP
	JWTSecret      []byte
	AuthClient     *authclient.Client
}

func Register(e *echo.Echo, d *Deps) {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	authMW := middleware.NewAutoRefreshMiddleware(d.JWTSecret, d.AuthClient)

	products := e.Group("/order")
	products.GET("", d.OrderHandler.GetOrder)
	products.GET("/:id", d.OrderHandler.GetOrders)
	products.POST("/create/:id", d.OrderHandler.CreateOrder)
	products.POST("/cancel/:id", d.OrderHandler.CancelOrder)

	admin := products.Group("", authMW.RequireAdmin)
	admin.PATCH("/:id", d.OrderHandler.UpdateOrder)
}
