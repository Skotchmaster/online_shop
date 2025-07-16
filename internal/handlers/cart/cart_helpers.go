package cart

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type Kafka struct {
	Producer *mykafka.Producer
}

func (h *Kafka) PublishEvent(c echo.Context, event map[string]interface{}, UserID uint) {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	if err := h.Producer.PublishEvent(
		ctx,
		"cart_events",
		fmt.Sprint(UserID),
		event,
	); err != nil {
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

func (h *CartHandler) SimulateDelivery(steps []DeliveryStep, orderID uint) {
	go func() {
		for _, step := range steps {
			time.Sleep(step.After)

			if err := h.DB.Model(&models.Order{}).Where("id=?", orderID); err != nil {
				log.Printf("failed to update status of order %d: %v", orderID, err)
			} else {
				log.Printf("order %d status set up %s", orderID, step.Status)
			}
		}
	}()
}
