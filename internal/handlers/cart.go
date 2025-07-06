package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type CartHandler struct {
	DB       *gorm.DB
	Producer *mykafka.Producer
}

func (h *CartHandler) GetCart(c echo.Context) error {
	UserID, err := GetID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid token")
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
	UserID, err := GetID(c)
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
		item.Quantity += req.Quantity
		h.DB.Save(&item)
	}

	if err != gorm.ErrRecordNotFound {
		return c.JSON(http.StatusBadRequest, err)
	}

	newitem := models.CartItem{
		UserID:    UserID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}

	if err := h.DB.Create(&newitem); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":      "add_cart_items",
		"UserID":    UserID,
		"cart_item": item,
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

	return c.JSON(http.StatusOK, newitem)
}

func (h *CartHandler) DeleteOneFromCart(c echo.Context) error {
	UserID, err := GetID(c)
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
	UserID, err := GetID(c)
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
