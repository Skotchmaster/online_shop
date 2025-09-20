package cart

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type Kafka struct {
	Producer *mykafka.Producer
}

func (h *CartHandler) publish(c echo.Context, event map[string]any) {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()
	if err := h.Producer.PublishEvent(ctx, "cart_events", fmt.Sprint(event["userID"]), event); err != nil {
		c.Logger().Errorf("Kafka publish error: %v", err)
	}
}

func GetID(c echo.Context, jwt_secret []byte) (uint, error) {
	cookie, err := c.Cookie("accessToken")
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "missing auth cookie")
	}
	tokenString := cookie.Value
	if tokenString == "" {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "empty token")
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwt_secret, nil
	})
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "invalid token: "+err.Error())
	}
	if !token.Valid {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
	}
	subRaw, ok := claims["sub"].(float64)
	if !ok {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid subject claim")
	}

	return uint(subRaw), nil
}

type DeliveryStep struct {
	Status string
	After  time.Duration
}
