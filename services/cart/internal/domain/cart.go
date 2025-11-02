package cart

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Skotchmaster/online_shop/internal/logging"
	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CartHandler struct {
	DB        *gorm.DB
	Producer  *mykafka.Producer
	JWTSecret []byte
}

func (h *CartHandler) GetCart(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "get_cart")
		
	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		l.Warn("get_cart_error", "status", 401, "reason", "unauthorized", "error", err)
		return c.JSON(http.StatusUnauthorized, "unauthorized")
	}

	var items []models.CartItem
	if err := h.DB.WithContext(ctx).Where("user_id=?", userID).Find(&items).Error; err != nil {
		l.Warn("get_cart_error", "status", 500, "reason", "db_error", "error", err)
		return c.JSON(http.StatusNotFound, "db_error")
	}

	event := map[string]interface{}{
		"type":   "get_cart",
		"userID": userID,
	}
	h.publish(c, event)
	l.Info("get_cart_success")
	return c.JSON(http.StatusOK, items)

}

func (h *CartHandler) AddToCart(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "add_to_cart")	
	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		l.Warn("add_to_cart_error", "status", 400, "reason", "invalid body", "error", err)
		return c.JSON(http.StatusBadRequest, "invalid body")
	}

	var req struct {
		Quantity  uint `json:"quantity"`
		ProductID uint `json:"product_id"`
	}
	if err := c.Bind(&req); err != nil {
		l.Warn("add_to_cart_error", "status", 400, "reason", "invalid body", "error", err)
		return c.JSON(http.StatusBadRequest, err)
	}

	if req.Quantity < 1 {
		req.Quantity = 1
	}

	var item models.CartItem

	if err := h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND product_id = ?", userID, req.ProductID).First(&item).Error

		switch{
		case errors.Is(err, gorm.ErrRecordNotFound):
			item = models.CartItem{
				UserID: userID,
				ProductID: req.ProductID,
				Quantity: req.Quantity,
			}
			if err := tx.Create(&item).Error; err != nil{
				return err
			}
			return nil
		case err == nil:
			if err := tx.Model(&item).Update("quantity", gorm.Expr("quantity + ?", req.Quantity)).Error; err != nil{
				return err
			}	
			item.Quantity += req.Quantity
			return nil
			default:
			return err
		}
	}); err != nil{
		l.Error("add_to_cart_error", "status", 500, "reason", "db error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "db error")
	}
	l.Info("add_to_cart_success")
	h.publish(c, map[string]any{
		"type":      "add_cart_items",
		"userID":    userID,
		"productID": req.ProductID,
		"quantity":  item.Quantity,
	})
	return c.JSON(http.StatusOK, item)
}

func (h *CartHandler) DeleteOneFromCart(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "delete_one_from_cart_cart")

	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		l.Warn("delete_one_from_cart_cart_error", "status", 401, "reason", "unauthorized", "error", err)
		return c.JSON(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		l.Warn("delete_one_from_cart_cart_error", "status", 400, "reason", "invalid id", "error", err)
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	var item models.CartItem
	deleted := false

	if err := h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", id, userID).First(&item).Error; err != nil {
			return err
		}
		if item.Quantity > 1 {
			if err := tx.Model(&item).Update("quantity", gorm.Expr("quantity - 1")).Error; err != nil {
				return err
			}
			item.Quantity --
		} else {
			if err := tx.Delete(&item).Error; err != nil {
				return err
			}
			deleted = true
		}
		return nil
	}); err != nil{
		if errors.Is(err, gorm.ErrRecordNotFound){
			l.Error("delete_one_from_cart_error", "status", 404, "reason", "record not found", "error", err)
			return echo.NewHTTPError(http.StatusNotFound, "record not found")
		}else{
			l.Error("delete_one_from_cart_error", "status", 500, "reason", "db error", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "db error")
		}
	}

	if deleted {
		h.publish(c, map[string]any{
		"type":         "cart_item_deleted",
		"userID":       userID,
		"deleted_item": id,
		})
		l.Info("delete_one_from_cart_success")
		return c.JSON(http.StatusOK, map[string]any{"deleted_item": id})
	} else {
		h.publish(c, map[string]any{
			"type":         "one_elem_deleted",
			"userID":       userID,
			"id":           item.ID,
			"new_quantity": item.Quantity,
		})
		l.Info("delete_one_from_cart_success")
		return c.JSON(http.StatusOK, item)
	}
}

func (h *CartHandler) DeleteAllFromCart(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "delete_all_from_cart_cart")

	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		l.Warn("delete_all_from_cart_cart_error", "status", 401, "reason", "unauthorized", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		l.Warn("delete_all_from_cart_cart_error", "status", 400, "reason", "invalid id", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var remaining []models.CartItem
	if err := h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CartItem{}) 
		if res.Error != nil {
			return res.Error
		}	
		if res.RowsAffected == 0{
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("user_id = ?", userID).Find(&remaining).Error; err != nil {
			return err
		}
		return nil
	}); err != nil{
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Warn("delete_all_from_cart_error", "status", 404, "reason", "record_not_found", "error", err)
			return c.JSON(http.StatusNotFound, "record not found")
		}
		l.Error("delete_all_from_cart_error", "status", 500, "reason", "db_error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "db error")
	}

	event := map[string]interface{}{
		"type":         "cart_item_deleted",
		"userID":       userID,
		"deleted_item": id,
		"remaining":    remaining,
	}
	h.publish(c, event)
	l.Info("delete_all_from_cart_success")
	return c.JSON(http.StatusOK, remaining)
}

func (h *CartHandler) MakeOrder(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "make_order")

	userID, err := GetID(c, h.JWTSecret)
	if err != nil {
		l.Warn("make_order_error", "status", 401, "reason", "unauthorized", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var (
		order      models.Order
		orderItems []models.OrderItem
	)

	txErr := h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var items []models.CartItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).Find(&items).Error; err != nil {
			l.Warn("make_order_error", "status", 500, "reason", "db_error", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "db_error")
		}
		if len(items) == 0 {
			l.Warn("make_order_error", "status", 400, "reason", "no items in cart", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "no items in cart")
		}

		var total float64
		for _, it := range items {
			var p models.Product
			if err := tx.First(&p, it.ProductID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					l.Warn("make_order_error", "status", 404, "reason", "product not found", "error", err)
					return echo.NewHTTPError(http.StatusNotFound, "product not found")
				}
				l.Error("make_order_error", "status", 500, "reason", "db_error", "error", err)
				return echo.NewHTTPError(http.StatusBadRequest, "db_error")
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
			l.Warn("make_order_error", "status", 500, "reason", "cannot create order", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "cannot create order")
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
				l.Warn("make_order_error", "status", 500, "reason", "cannot create order item", "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "cannot create order item")
			}
		}

		if err := tx.Where("user_id = ?", userID).Delete(&models.CartItem{}).Error; err != nil {
			l.Warn("make_order_error", "status", 500, "reason", "cannot delete cart item", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "cannot delete cart item")
		}
		return nil
	})

	if txErr != nil {
		if he, ok := txErr.(*echo.HTTPError); ok {
			return he
		}
		l.Error("make_order_error", "status", 500, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "db error")
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
	l.Info("make_order_success")
	return c.JSON(http.StatusOK, resp)
}
