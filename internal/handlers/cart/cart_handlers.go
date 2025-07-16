package cart

import (
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
	Kafka := Kafka{Producer: &mykafka.Producer{}}
	Kafka.PublishEvent(c, event, UserID)

	return c.JSON(http.StatusOK, items)

}

func (h *CartHandler) AddToCart(c echo.Context) error {
	UserID, err := GetID(c, h.JWTSecret)
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

	newItem := models.CartItem{
		UserID:    UserID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}
	if err := h.DB.Create(&newItem).Error; err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":      "add_cart_items",
		"UserID":    UserID,
		"cart_item": newItem,
	}
	Kafka := Kafka{Producer: &mykafka.Producer{}}
	Kafka.PublishEvent(c, event, UserID)

	return c.JSON(http.StatusOK, newItem)
}

func (h *CartHandler) DeleteOneFromCart(c echo.Context) error {
	UserID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid token")
	}

	var item models.CartItem
	if err := h.DB.Where("user_id=?", UserID).First(&item).Error; err == nil {
		if item.Quantity > 1 {
			item.Quantity -= 1
			if err := h.DB.Save(&item).Error; err != nil {
				return echo.NewHTTPError(500, err.Error())
			}
			return c.JSON(http.StatusOK, item)
		} else {
			return h.DeleteAllFromCart(c)
		}
	}

	event := map[string]interface{}{
		"type":         "delete_cart_item",
		"UserID":       UserID,
		"new_quantity": item.Quantity,
	}
	Kafka := Kafka{Producer: &mykafka.Producer{}}
	Kafka.PublishEvent(c, event, UserID)

	return c.JSON(http.StatusOK, item)
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
	Kafka := Kafka{Producer: &mykafka.Producer{}}
	Kafka.PublishEvent(c, event, UserID)

	return c.JSON(http.StatusOK, remaining)
}

func (h *CartHandler) MakeOrder(c echo.Context) error {
	UserID, err := GetID(c, h.JWTSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	var Order models.Order
	var OrderItems []models.OrderItem
	new_err := h.DB.Transaction(func(tx *gorm.DB) error {
		var items []models.CartItem
		if err := tx.Where("user_id=?", UserID).Find(&items).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		if len(items) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "no items in cart")
		}

		total := float64(0)
		for _, item := range items {
			var product models.Product
			if err := tx.Where("ID=?", item.ProductID).First(&product).Error; err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err)
			}
			total += float64(item.Quantity) * product.Price
		}

		Order = models.Order{
			UserID:    UserID,
			CreatedAt: time.Now().Unix(),
			Total:     total,
			Status:    "new",
		}
		if err := tx.Create(&Order).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		for _, item := range items {
			OrderItem := models.OrderItem{
				OrderID:   Order.ID,
				UserID:    UserID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			}
			OrderItems = append(OrderItems, OrderItem)
			if err := tx.Create(&OrderItem).Error; err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err)
			}
		}
		if err := tx.Where("user_id=?", UserID).Delete(&models.CartItem{}).Error; err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		return nil
	})
	if new_err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, new_err.Error())
	}

	event := map[string]interface{}{
		"type":     "order_created",
		"user_id":  UserID,
		"order_id": Order.ID,
		"order":    Order,
		"items":    OrderItems,
	}
	Kafka := Kafka{Producer: &mykafka.Producer{}}
	Kafka.PublishEvent(c, event, UserID)

	type OrderResponse struct {
		OrderID uint               `json:"order_id"`
		Total   float64            `json:"total"`
		Status  string             `json:"status"`
		Items   []models.OrderItem `json:"items"`
	}

	Response := OrderResponse{
		OrderID: Order.ID,
		Total:   Order.Total,
		Status:  Order.Status,
		Items:   OrderItems,
	}

	steps := []DeliveryStep{
		{"processing", 1 * time.Minute},
		{"shipped", 2 * time.Minute},
		{"delivered", 3 * time.Minute},
	}
	h.SimulateDelivery(steps, Order.ID)

	return c.JSON(http.StatusOK, Response)

}
