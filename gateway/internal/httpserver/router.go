package httpserver

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/gateway/internal/middleware"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	AuthURL string
	CartURL string
	CatalogURL string
	OrderURL string

	JWTSecret []byte
}

func Register(e *echo.Echo, d *Deps) error {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	for _, m := range middleware.Common() {
		e.Use(m)
	}

	authProxy, err := newProxy(d.AuthURL, "/api/v1/auth")
	if err != nil {
		return err
	}

	catalogProxy, err := newProxy(d.CatalogURL, "/api/v1")
	if err != nil {
		return err
	}

	orderProxy, err := newProxy(d.OrderURL, "/api/v1")
	if err != nil {
		return err
	}

	cartProxy, err := newProxy(d.CartURL, "/api/v1")
	if err != nil {
		return err
	}

	e.Any("/api/v1/auth/*", authProxy)
	e.Match([]string{http.MethodGet}, "/api/v1/catalog/*", catalogProxy)

	api := e.Group("/api/v1")
	api.Use(middleware.Middleware(d.JWTSecret))

	api.Match([]string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}, "/catalog", catalogProxy)
	api.Match([]string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}, "/catalog/*", catalogProxy)
	api.Any("/cart", cartProxy)
	api.Any("/cart/*", cartProxy)
	api.Any("/orders", orderProxy)
	api.Any("/orders/*", orderProxy)

	return nil
}
