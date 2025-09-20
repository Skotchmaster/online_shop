package httpserver

import (
	"net/http"
	"time"

	"github.com/Skotchmaster/online_shop/internal/handlers"
	authhdl "github.com/Skotchmaster/online_shop/internal/handlers/auth"
	"github.com/Skotchmaster/online_shop/internal/handlers/cart"
	"github.com/Skotchmaster/online_shop/internal/middleware/auth"
	csrfmw "github.com/Skotchmaster/online_shop/internal/middleware/csrf"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type Deps struct {
	DB             *gorm.DB
	ProductHandler *handlers.ProductHandler
	AuthHandler    *authhdl.AuthHandler
	CartHandler    *cart.CartHandler
	ServiceHandler *auth.TokenService
	SearchHandler  *handlers.SearchHandler
}

func Register(e *echo.Echo, d *Deps) {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(200) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(200) })

	v1 := e.Group("/api/v1")

	csrfCfg := csrfmw.Config{
    CookieName:        "XSRF-TOKEN",
    HeaderName:        "X-CSRF-Token",
    FormField:         "csrf_token",
    CookiePath:        "/api/v1",
    Domain:            "",
    Secure:            false,                    // true в проде
    SameSite:          http.SameSiteLaxMode,
    MaxAge:            24 * time.Hour,
    EnforceSameOrigin: true,
    SkipPaths: nil,
}
	v1.Use(csrfmw.Middleware(csrfCfg))

	v1.POST("/register", d.AuthHandler.Register)
	v1.POST("/login", d.AuthHandler.Login)
	v1.POST("/logout", d.AuthHandler.LogOut)
	v1.GET("/search", d.SearchHandler.Search)

	admin := v1.Group("/admin", d.ServiceHandler.AutoRefreshMiddlewareAdmin)

	admin.POST("/products", d.ProductHandler.CreateProduct)
	admin.PATCH("/products/:id", d.ProductHandler.PatchProduct)
	admin.DELETE("/products/:id", d.ProductHandler.DeleteProduct)

	products := v1.Group("/products")

	products.GET("/:id", d.ProductHandler.GetProduct)
	products.GET("", d.ProductHandler.GetProducts)

	cart := v1.Group("cart", d.ServiceHandler.AutoRefreshMiddleware)

	cart.GET("", d.CartHandler.GetCart)
	cart.POST("", d.CartHandler.AddToCart)
	cart.POST("/order", d.CartHandler.MakeOrder)
	cart.DELETE("/:id", d.CartHandler.DeleteOneFromCart)
	cart.DELETE("/:id/all", d.CartHandler.DeleteAllFromCart)

}
