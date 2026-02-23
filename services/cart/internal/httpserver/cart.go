package httpserver

import (
	"errors"
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/logging"
	"github.com/Skotchmaster/online_shop/services/cart/internal/models"
	"github.com/Skotchmaster/online_shop/services/cart/internal/service"
	"github.com/Skotchmaster/online_shop/services/cart/internal/transport"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CartHTTP struct {
	Svc *service.CartService
}

func (h *CartHTTP) GetID(c echo.Context) (uuid.UUID, error) {
    v := c.Get("user_id")
    s, ok := v.(string)
    if !ok || s == "" {
        return uuid.Nil, errors.New("unauthorized")
    }

    userID, err := uuid.Parse(s)
    if err != nil {
        return uuid.Nil, errors.New("unauthorized")
    }
    
    return userID, nil
}

func (h *CartHTTP) GetCart(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "get.cart")
	
	userID, err := h.GetID(c)
	if err != nil{
		l.Error("get_cart_error", "status", 401, "reason", "unauthorized", "error", err)
		return c.JSON(http.StatusUnauthorized, "unauthorized")
	}

	items, err := h.Svc.GetCart(ctx, userID)
	if err != nil {
		l.Error("get_cart_error", "status", 500, "reason", "internal server error", "error", err)
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	l.Info("cart successfully got")
	return c.JSON(http.StatusOK, items)
}

func (h *CartHTTP) AddToCart(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "add.cart")

	userID, err := h.GetID(c)
	if err != nil {
		l.Error("add_cart_error", "status", 401, "reason", "unauthorized", "error", err)
		return c.JSON(http.StatusUnauthorized, "unauthorized")
	}

	var req struct {
		Quantity  uint `json:"quantity"`
		ProductID uuid.UUID `json:"product_id"`
	}

	if err := c.Bind(&req); err != nil {
		l.Warn("add_to_cart_error", "status", 400, "reason", "invalid body", "error", err)
		return c.JSON(http.StatusBadRequest, "invalid body")
	}

	item := models.CartItem{
		UserID: userID,
		ProductID: req.ProductID,
		Quantity: req.Quantity,
	}
	if err := h.Svc.AddToCart(ctx, &item); err != nil {
		if errors.Is(err, service.ErrValidation) {
			l.Warn("add_to_cart_error", "status", 400, "reason", "invalid body", "error", err)
			return c.JSON(http.StatusBadRequest, "invalid body")
		}
		l.Error("add_to_cart_error", "status", 500, "reason", "internal error", "error", err)
		return c.JSON(http.StatusInternalServerError, "internal error")
	}

	l.Info("item added successfully to cart")
	return c.JSON(http.StatusCreated, item)
}

func (h *CartHTTP) DeleteOneFromCart(c echo.Context) error {
    ctx := c.Request().Context()
    l := logging.FromContext(ctx).With("handler", "delete.one.from.cart")

    userID, err := h.GetID(c)
    if err != nil {
        l.Error("delete_one_from_cart_error", "status", 401, "reason", "unauthorized", "error", err)
        return c.JSON(http.StatusUnauthorized, "unauthorized")
    }

    var req struct {
        ProductID uuid.UUID `json:"product_id"`
    }
    if err := c.Bind(&req); err != nil {
        l.Warn("delete_one_from_cart_error", "status", 400, "reason", "invalid body", "error", err)
        return c.JSON(http.StatusBadRequest, "invalid body")
    }

    deleted, item, err := h.Svc.DeleteOneFromCart(ctx, req.ProductID, userID)
    if err != nil {
        if errors.Is(err, service.ErrNotFound) {
            l.Warn("delete_one_from_cart_error", "status", 404, "reason", "item not found", "error", err)
            return c.JSON(http.StatusNotFound, "item not found")
        } else {
			if errors.Is(err, service.ErrValidation) {
            l.Warn("delete_one_from_cart_error", "status", 400, "reason", "invalid body", "error", err)
            return c.JSON(http.StatusBadRequest, "invalid body")
			}
		}
        l.Error("delete_one_from_cart_error", "status", 500, "reason", "internal error", "error", err)
        return c.JSON(http.StatusInternalServerError, "internal error")
    }

	var resp transport.DeleteOneFromCartResponse
	
	if deleted {
		resp.ProductID = req.ProductID
		resp.Deleted = deleted
		resp.Quantity = 0
	} else {
		resp.ProductID = item.ProductID
		resp.Deleted = deleted
		resp.Quantity = item.Quantity
	}
    return c.JSON(http.StatusOK, resp)
}


func (h *CartHTTP) DeleteAllFromCart(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "delete.all.from.cart")

	userID, err := h.GetID(c)
	if err != nil {
		l.Error("delete_all_from_cart_error", "status", 401, "reason", "unauthorized", "error", err)
		return c.JSON(http.StatusUnauthorized, "unauthorized")
	}

	if err := h.Svc.DeleteAllFromCart(ctx, userID); err != nil {
		l.Error("delete_all_from_cart_cart_error", "status", 500, "reason", "internal error", "error", err)
		return c.JSON(http.StatusInternalServerError, "internal error")
	}

	l.Info("cart successfully cleared")
	return c.JSON(http.StatusOK, "cart successfully cleared")
}