package httpserver

import (
	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/Skotchmaster/online_shop/internal/handlers/cart"
	"github.com/Skotchmaster/online_shop/internal/service/token"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type Deps struct {
	DB             *gorm.DB
	ProductHandler *handlers.ProductHandler
	AuthHandler    *handlers.AuthHandler
	CartHandler    *cart.CartHandler
	ServiceHandler *token.TokenService
	SearchHandler  *handlers.SearchHandler
}

func Register(e *echo.Echo, d *Deps) {
	e.GET("/health/live", func(c echo.Context) error { return c.NoContent(200) })
	e.GET("/health/ready", func(c echo.Context) error { return c.NoContent(200) })

	v1 := e.Group("/api/v1")

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
