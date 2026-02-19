package httpserver

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/authclient"
	middleware "github.com/Skotchmaster/online_shop/pkg/middleware/auth"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	CatalogHandler *CatalogHTTP
	JWTSecret      []byte
	AuthClient     *authclient.Client
}

func Register(e *echo.Echo, d *Deps) {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	authMW := middleware.NewAutoRefreshMiddleware(d.JWTSecret, d.AuthClient)

	products := e.Group("/catalog/products")
	products.GET("/search", d.CatalogHandler.SearchProducts)
	products.GET("", d.CatalogHandler.GetProducts)
	products.GET("/:id", d.CatalogHandler.GetProduct)

	admin := products.Group("", authMW.RequireAdmin)
	admin.POST("", d.CatalogHandler.CreateProduct)
	admin.PATCH("/:id", d.CatalogHandler.PatchProduct)
	admin.DELETE("/:id", d.CatalogHandler.DeleteProduct)
}
