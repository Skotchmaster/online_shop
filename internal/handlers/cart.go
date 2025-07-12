package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type CartHandler struct {
	DB        *gorm.DB
	Producer  *mykafka.Producer
	JWTSecret []byte
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

func (h *CartHandler) GetCart(c echo.Context) error {
	UserID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return err
	}

	var items []models.CartItem
	if err := h.DB.Where("user_id=?", UserID).Find(&items).Error; err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":       "get_cart_items",
		"UserID":     UserID,
		"cart_items": items,
	}
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

	return c.JSON(http.StatusOK, items)

}

func (h *CartHandler) AddToCart(c echo.Context) error {
	UserID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return err
	}

	var req struct {
		Quantity  uint
		ProductID uint
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if req.Quantity < 1 {
		req.Quantity = 1
	}

	newItem := models.CartItem{
		UserID:    UserID,
		ProductID: req.ProductID,
		Quantity:  req.ProductID,
	}
	if err := h.DB.Create(&newItem).Error; err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":      "add_cart_items",
		"UserID":    UserID,
		"cart_item": newItem,
	}

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

	return c.JSON(http.StatusOK, newItem)
}

func (h *CartHandler) DeleteOneFromCart(c echo.Context) error {
	UserID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid token")
	}
	var req struct {
		Quantity  uint
		ProductID uint
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if req.Quantity < 1 {
		req.Quantity = 1
	}

	var item models.CartItem
	if err := h.DB.Where("user_id=? AND product_id=?", UserID, req.ProductID).First(&item).Error; err == nil {
		if item.Quantity > req.Quantity {
			item.Quantity -= req.Quantity
			h.DB.Save(&item)
		} else {
			h.DeleteAllFromCart(c)
		}

	}

	event := map[string]interface{}{
		"type":          "delete_cart_item",
		"UserID":        UserID,
		"deleted_items": item,
	}

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

	return c.NoContent(http.StatusNoContent)
}

func (h *CartHandler) DeleteAllFromCart(c echo.Context) error {
	UserID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return err
	}

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := h.DB.
		Where("id = ? AND user_id = ?", id, UserID).
		Delete(&models.CartItem{}).Error; err != nil {

		return c.JSON(http.StatusInternalServerError, err)
	}

	var remaining []models.CartItem
	if err := h.DB.Where("user_id = ?", UserID).Find(&remaining).Error; err != nil {
		c.Logger().Errorf("DB read after delete error: %v", err)
	}

	event := map[string]interface{}{
		"type":         "cart_item_deleted",
		"user_id":      UserID,
		"deleted_item": id,
		"remaining":    remaining,
	}
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

	return c.NoContent(http.StatusNoContent)
}
