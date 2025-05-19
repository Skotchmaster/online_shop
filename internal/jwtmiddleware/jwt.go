package jwtmiddleware

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func JWTMiddleware(keyStore map[string][]byte) echo.MiddlewareFunc {
	cfg := echojwt.WithConfig(echojwt.Config{
		SigningMethod: "HS256",
		ContextKey:    "user",
		TokenLookup:   "header:Authorization",
		KeyFunc: func(token *jwt.Token) (interface{}, error) {
			kidVal, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("JWT token missing kid header")
			}
			secret, exists := keyStore[kidVal]
			if !exists {
				return nil, fmt.Errorf("unknown kid %s", kidVal)
			}
			return secret, nil
		},
	})
	return cfg
}
