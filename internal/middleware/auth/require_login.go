package auth

import (
	"time"

	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func (t *TokenService) AutoRefreshMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		newAccess, newRefresh, _, err := t.CheckCookie(c)
		if err != nil {
			return err
		}

		if newRefresh == "" {
			return next(c)
		}

		c.SetCookie(handlers.CreateCookie("accessToken", newAccess, "/", time.Now().Add(15*time.Minute)))
		c.SetCookie(handlers.CreateCookie("refreshToken", newRefresh, "/", time.Now().Add(7*24*time.Hour)))

		token, _ := jwt.Parse(newAccess, func(j *jwt.Token) (interface{}, error) { return t.JWTSecret, nil })
		setUserContext(c, token.Claims.(jwt.MapClaims))
		return next(c)
	}
}
