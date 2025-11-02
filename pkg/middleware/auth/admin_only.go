package auth

import (
	"fmt"
	"time"

	authhdl "github.com/Skotchmaster/online_shop/internal/handlers/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func (t *TokenService) AutoRefreshMiddlewareAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		newAccess, newRefresh, role, err := t.CheckCookie(c)
		if err != nil {
			return err
		}

		if newRefresh == "" {
			if role == "admin" {
				token, _ := jwt.Parse(newAccess, func(j *jwt.Token) (interface{}, error) { return t.JWTSecret, nil })
				setUserContext(c, token.Claims.(jwt.MapClaims))
				return next(c)
			}
			return fmt.Errorf("you don't have enough rights")
		}

		if role == "admin" {
			c.SetCookie(authhdl.CreateCookie("accessToken", newAccess, "/", time.Now().Add(15*time.Minute)))
			c.SetCookie(authhdl.CreateCookie("refreshToken", newRefresh, "/", time.Now().Add(7*24*time.Hour)))

			token, _ := jwt.Parse(newAccess, func(j *jwt.Token) (interface{}, error) { return t.JWTSecret, nil })
			setUserContext(c, token.Claims.(jwt.MapClaims))
			return next(c)
		}
		return fmt.Errorf("you don't have enough rights")
	}
}