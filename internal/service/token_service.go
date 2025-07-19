package service

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type TokenService struct {
	DB            *gorm.DB
	RefreshSecret []byte
	JWTSecret     []byte
}

func (t *TokenService) CheckCookie(c echo.Context) (string, string, string, error) {
	asCookie, err := c.Cookie("accessToken")
	if err == nil {
		token, err := jwt.Parse(asCookie.Value, func(j *jwt.Token) (interface{}, error) {
			return t.JWTSecret, nil
		})
		if err == nil && token.Valid {
			claims := token.Claims.(jwt.MapClaims)
			role, ok := claims["role"].(string)
			if !ok {
				return "", "", "", echo.NewHTTPError(http.StatusForbidden, "not enough rights")
			}
			setUserContext(c, token.Claims.(jwt.MapClaims))
			return asCookie.Value, "", role, nil
		}
		if errors.Is(err, jwt.ErrTokenExpired) {
		} else {
			return "", "", "", echo.NewHTTPError(http.StatusUnauthorized, err)
		}
	}

	rfCookie, err := c.Cookie("refreshToken")
	if err != nil {
		return "", "", "", err
	}
	newAccess, newRefresh, claims, err := t.RotateToken(rfCookie.Value)
	if err != nil {
		return "", "", "", err
	}

	role, ok := claims["role"].(string)
	if !ok {
		return "", "", "", echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	return newAccess, newRefresh, role, nil
}

func (t *TokenService) RotateToken(rawToken string) (string, string, jwt.MapClaims, error) {
	claims, err := ValidateRefresh(rawToken, t.RefreshSecret, t.DB)
	if err != nil {
		return "", "", nil, err
	}

	userID := uint(claims["sub"].(float64))
	role := claims["role"].(string)

	newAccess, err := SingAccessToken(userID, role, t.JWTSecret)
	if err != nil {
		return "", "", nil, err
	}

	newRefresh, err := SingRefreshToken(userID, role, t.RefreshSecret)
	if err == nil {
		return "", "", nil, err
	}

	if err := SaveRefreshToken(t.DB, newRefresh, userID); err != nil {
		return "", "", nil, err
	}

	return newAccess, newRefresh, claims, nil

}

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
			c.SetCookie(handlers.CreateCookie("accessToken", newAccess, "/", time.Now().Add(15*time.Minute)))
			c.SetCookie(handlers.CreateCookie("refreshToken", newRefresh, "/", time.Now().Add(7*24*time.Hour)))

			token, _ := jwt.Parse(newAccess, func(j *jwt.Token) (interface{}, error) { return t.JWTSecret, nil })
			setUserContext(c, token.Claims.(jwt.MapClaims))
			return next(c)
		}
		return fmt.Errorf("you don't have enough rights")
	}
}

func setUserContext(c echo.Context, claims jwt.MapClaims) {
	c.Set("userID", uint(claims["sub"].(float64)))
	c.Set("role", claims["role"].(string))
}
