package cart

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type CartHandler struct {
	DB        *gorm.DB
	Producer  *mykafka.Producer
	JWTSecret []byte
}

func (h *CartHandler) GetCart(c echo.Context) error {
	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return err
	}

	var items []models.CartItem
	if err := h.DB.Where("user_id=?", userID).Find(&items).Error; err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":   "get_cart",
		"userID": userID,
	}
	h.publish(c, event)

	return c.JSON(http.StatusOK, items)

}

func (h *CartHandler) AddToCart(c echo.Context) error {
	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return err
	}

	var req struct {
		Quantity  uint `json:"quantity"`
		ProductID uint `json:"product_id"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if req.Quantity < 1 {
		req.Quantity = 1
	}

	var item models.CartItem
	tx := h.DB.Where("user_id = ? AND product_id = ?", userID, req.ProductID).First(&item)
	if tx.Error == nil {
		item.Quantity += req.Quantity
		if err := h.DB.Save(&item).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		h.publish(c, map[string]any{
			"type":      "add_cart_items",
			"userID":    userID,
			"productID": req.ProductID,
			"quantity":  item.Quantity,
		})
		return c.JSON(http.StatusOK, item)
	}
	if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return echo.NewHTTPError(http.StatusInternalServerError, tx.Error.Error())
	}
	newItem := models.CartItem{
		UserID:    userID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}
	if err := h.DB.Create(&newItem).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	h.publish(c, map[string]any{
		"type":      "add_cart_items",
		"userID":    userID,
		"productID": req.ProductID,
		"quantity":  newItem.Quantity,
	})
	return c.JSON(http.StatusOK, newItem)
}

func (h *CartHandler) DeleteOneFromCart(c echo.Context) error {
	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, "invalid token")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	var item models.CartItem
	if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, "item not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if item.Quantity > 1 {
		item.Quantity -= 1
		if err := h.DB.Save(&item).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		h.publish(c, map[string]any{
			"type":         "one_elem_deleted",
			"userID":       userID,
			"id":           item.ID,
			"new_quantity": item.Quantity,
		})
		return c.JSON(http.StatusOK, item)
	}

	if err := h.DB.Delete(&item).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	h.publish(c, map[string]any{
		"type":         "cart_item_deleted",
		"userID":       userID,
		"deleted_item": id,
	})
	return c.JSON(http.StatusOK, map[string]any{"deleted_item": id})
}

func (h *CartHandler) DeleteAllFromCart(c echo.Context) error {
	userID, err := GetID(c, h.JWTSecret)
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

	var remaining []models.CartItem
	if err := h.DB.Where("user_id = ?", userID).Find(&remaining).Error; err != nil {
		c.Logger().Errorf("DB read after delete error: %v", err)
	}

	event := map[string]interface{}{
		"type":         "cart_item_deleted",
		"userID":       userID,
		"deleted_item": id,
		"remaining":    remaining,
	}
	h.publish(c, event)

	return c.JSON(http.StatusOK, remaining)
}

func (h *CartHandler) MakeOrder(c echo.Context) error {
	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	var (
		order      models.Order
		orderItems []models.OrderItem
	)

	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		var items []models.CartItem
		if err := tx.Where("user_id = ?", userID).Find(&items).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if len(items) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "no items in cart")
		}

		var total float64
		for _, it := range items {
			var p models.Product
			if err := tx.First(&p, it.ProductID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return echo.NewHTTPError(http.StatusBadRequest, "product not found")
				}
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			total += float64(it.Quantity) * p.Price
		}

		order = models.Order{
			UserID:    userID,
			Total:     total,
			Status:    "new",
			CreatedAt: time.Now().Unix(),
		}
		if err := tx.Create(&order).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		orderItems = make([]models.OrderItem, 0, len(items))
		for _, it := range items {
			oi := models.OrderItem{
				OrderID:   order.ID,
				UserID:    userID,
				ProductID: it.ProductID,
				Quantity:  it.Quantity,
			}
			orderItems = append(orderItems, oi)
			if err := tx.Create(&oi).Error; err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}

		if err := tx.Where("user_id = ?", userID).Delete(&models.CartItem{}).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return nil
	})

	if txErr != nil {
		if he, ok := txErr.(*echo.HTTPError); ok {
			return he
		}
		return echo.NewHTTPError(http.StatusInternalServerError, txErr.Error())
	}

	h.publish(c, map[string]any{
		"type":    "order_created",
		"userID":  userID,
		"orderID": order.ID,
		"items":   orderItems,
	})
	type OrderResponse struct {
		OrderID uint               `json:"order_id"`
		Total   float64            `json:"total"`
		Status  string             `json:"status"`
		Items   []models.OrderItem `json:"items"`
	}
	resp := OrderResponse{
		OrderID: order.ID,
		Total:   order.Total,
		Status:  order.Status,
		Items:   orderItems,
	}
	return c.JSON(http.StatusOK, resp)
}
