package handlers

import (
	"net/http"
	"strconv"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type CartHandler struct {
	DB *gorm.DB
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
	return c.NoContent(http.StatusNoContent)
}

func (h *CartHandler) DeleteAllFromCart(c echo.Context) error {
	userID, err := GetID(c)
	if err != nil {
		return err
	}

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if err := h.DB.
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.CartItem{}).Error; err != nil {

		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusNoContent)
}
