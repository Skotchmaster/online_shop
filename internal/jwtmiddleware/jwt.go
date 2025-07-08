package jwtmiddleware

import (
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func JWTMiddleware(secret []byte) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningMethod: "HS256",
		SigningKey:    secret,
		ContextKey:    "user",
		TokenLookup:   "header:Authorization",
	})
}
