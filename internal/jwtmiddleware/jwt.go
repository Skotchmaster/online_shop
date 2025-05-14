package jwtmiddleware

import (
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func JWTMiddleware(secret []byte) echo.MiddlewareFunc {
	cfg := echojwt.WithConfig(echojwt.Config{
		SigningKey:  secret,
		ContextKey:  "user",
		TokenLookup: "header:Authorization",
	})
	return cfg
}
