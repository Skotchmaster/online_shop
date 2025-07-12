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

func (t *TokenService) RotateToken(rawToken string) (string, string, error) {
	claims, err := ValidateRefresh(rawToken, t.RefreshSecret, t.DB)
	if err != nil {
		return "", "", err
	}

	userID := uint(claims["sub"].(float64))
	role := claims["role"].(string)

	newAccess, err := SingAccessToken(userID, role, t.JWTSecret)
	if err != nil {
		return "", "", err
	}

	newRefresh, err := SingRefreshToken(userID, role, t.RefreshSecret)
	if err == nil {
		return "", "", err
	}

	if err := SaveRefreshToken(t.DB, newRefresh, userID); err != nil {
		return "", "", err
	}

	return newAccess, newRefresh, nil

}

func (t *TokenService) RevokeRefresh(DB *gorm.DB, token string) error {

	if err := DB.Where("token=?", token).Update("revoked", true).Error; err != nil {
		return fmt.Errorf("DB error: %w", err)
	}

	return nil
}

func (t *TokenService) AutoRefreshMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		asCookie, err := c.Cookie("accessToken")
		if err == nil {
			token, err := jwt.Parse(asCookie.Value, func(j *jwt.Token) (interface{}, error) {
				return t.JWTSecret, nil
			})
			if err == nil && token.Valid {
				setUserContext(c, token.Claims.(jwt.MapClaims))
				return next(c)
			}
			if errors.Is(err, jwt.ErrTokenExpired) {
			} else {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid access token")
			}
		}

		rfCookie, err := c.Cookie("refreshToken")
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "refresh token missing")
		}
		newAccess, newRefresh, err := t.RotateToken(rfCookie.Value)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "cannot rotate token: "+err.Error())
		}

		c.SetCookie(handlers.CreateCookie("accessToken", newAccess, "/", time.Now().Add(15*time.Minute)))
		c.SetCookie(handlers.CreateCookie("refreshToken", newRefresh, "/", time.Now().Add(7*24*time.Hour)))

		token, _ := jwt.Parse(newAccess, func(j *jwt.Token) (interface{}, error) { return t.JWTSecret, nil })
		setUserContext(c, token.Claims.(jwt.MapClaims))
		return next(c)
	}
}

func setUserContext(c echo.Context, claims jwt.MapClaims) {
	c.Set("userID", uint(claims["sub"].(float64)))
	c.Set("role", claims["role"].(string))
}
