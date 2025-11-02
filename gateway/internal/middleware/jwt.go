package middleware

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const (
	CtxUserID = "user_id"
	CtxRole   = "role"
)

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func Middleware(secret []byte) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing Authorization header")
			}
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid Authorization header")
			}
			tokenStr := parts[1]

			claims := &Claims{}
			tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, errors.New("unexpected sign method")
				}
				return secret, nil
			})

			if err != nil || !tkn.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
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
	return func (next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, _ := c.Get(CtxRole).(string)
			if role == ""{
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or missing role")
			}
			if !slices.Contains(required, role) {
				return echo.NewHTTPError(http.StatusForbidden, "you don't have enough rights to see this page")
			}
			return next(c)
		}
	}
}
