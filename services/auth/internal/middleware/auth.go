package middleware

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/tokens"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/labstack/echo/v4"
)

type SimpleAuth struct {
	JWTSecret []byte
}

func NewSimpleAuth(secret []byte) *SimpleAuth {
	return &SimpleAuth{JWTSecret: secret}
}

func (m *SimpleAuth) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		accessCookie, err := c.Cookie("accessToken")
		if err != nil || accessCookie.Value == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing access token")
		}

		claims, err := tokens.AccessClaimsFromToken(accessCookie.Value, m.JWTSecret)
		if err != nil || claims == nil {
			c.SetCookie(jwthelp.DeleteCookie("accessToken", "/"))
			c.SetCookie(jwthelp.DeleteCookie("refreshToken", "/"))
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
		}

		c.Set("user_id", claims.Subject)
		c.Set("role", claims.Role)

		return next(c)
	}
}