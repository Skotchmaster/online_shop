package middleware

import (
	"net/http"
	"slices"

	"github.com/Skotchmaster/online_shop/pkg/tokens"
	"github.com/labstack/echo/v4"
)

const (
	CtxUserID = "user_id"
	CtxRole   = "role"
)

func Middleware(secret []byte) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			accessCookie, err := c.Cookie("accessToken")
			if err != nil || accessCookie.Value == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing access token")
			}
			claims, err := tokens.AccessClaimsFromToken(accessCookie.Value, secret)
			if err != nil || claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			if claims.Subject == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "token has no subject")
			}
			c.Set(CtxUserID, claims.Subject)
			c.Set(CtxRole, claims.Role)

			return next(c)
		}
	}
}

func RequireRole(required []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, _ := c.Get(CtxRole).(string)
			if role == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or missing role")
			}
			if !slices.Contains(required, role) {
				return echo.NewHTTPError(http.StatusForbidden, "you don't have enough rights to see this page")
			}
			return next(c)
		}
	}
}