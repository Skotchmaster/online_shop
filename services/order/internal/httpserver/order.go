package httpserver

import (
	"errors"
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/logging"
	"github.com/Skotchmaster/online_shop/pkg/util"
	"github.com/Skotchmaster/online_shop/services/order/internal/service"
	"github.com/Skotchmaster/online_shop/services/order/internal/transport"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type OrderHTTP struct {
	Svc *service.OrderService
}

func (h *OrderHTTP) GetID(c echo.Context) (uuid.UUID, error) {
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

func(h *OrderHTTP) CreateOrder(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "order.create_order")

	userID, err := h.GetID(c)
	if err != nil {
		l.Warn("create_order_error", "status", 401, "reason", "unauthorized", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	var req transport.CreateOrderRequest

	if err := c.Bind(&req); err != nil {
		l.Warn("create_order_error", "status", 400, "reason", "invalid body", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	order, err := h.Svc.CreateOrder(ctx, req, userID)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			l.Warn("create_order_error", "status", 400, "reason", "invalid body", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
		} else {
			l.Warn("create_order_error", "status", 500, "reason", "internal error", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	}

	l.Info("create_order_success")
	return c.JSON(http.StatusCreated, order)
}

func(h *OrderHTTP) GetOrders(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "order.create_order")

	userID, err := h.GetID(c)
	if err != nil {
		l.Warn("create_order_error", "status", 401, "reason", "unauthorized", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	page := util.ParseIntDefault(c.QueryParam("page"), 1)
    size := util.ParseIntDefault(c.QueryParam("size"), util.DefaultPageSize)

    offset, limit := util.Calculate(page, size)

    orders, err := h.Svc.ListOrders(ctx, userID, limit, offset)
	if err != nil {
		l.Error("create_order_error", "status", 500, "reason", "internal server error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	l.Info("get_orders_success")
	return c.JSON(http.StatusOK, orders)

}