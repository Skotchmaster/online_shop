package httpserver

import (
	"errors"
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/logging"
	"github.com/Skotchmaster/online_shop/pkg/util"
	"github.com/Skotchmaster/online_shop/services/order/internal/models"
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
	l := logging.FromContext(ctx).With("handler", "order.get_orders")

	userID, err := h.GetID(c)
	if err != nil {
		l.Warn("get_orders_error", "status", 401, "reason", "unauthorized", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	page := util.ParseIntDefault(c.QueryParam("page"), 1)
    size := util.ParseIntDefault(c.QueryParam("size"), util.DefaultPageSize)

    offset, limit := util.Calculate(page, size)

    orders, err := h.Svc.ListOrders(ctx, userID, limit, offset)
	if err != nil {
		l.Error("get_orders_error", "status", 500, "reason", "internal server error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	l.Info("get_orders_success")
	return c.JSON(http.StatusOK, orders)

}

func(h *OrderHTTP) GetOrder(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "order.get_order")

	idstr := c.Param("id")
	id, err := uuid.Parse(idstr)
	if err != nil {
		l.Error("get_order_error", "status", 400, "reason", "id is not uuid", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "id is not uuid")
	}

	userID, err := h.GetID(c)
	if err != nil {
		l.Warn("get_orders_error", "status", 401, "reason", "unauthorized", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}	

	order, err := h.Svc.GetOrder(ctx, id, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound){
			l.Error("get_order_error", "status", 404, "reason", "order not found", "error", err)
			return echo.NewHTTPError(http.StatusNotFound, "order not found")
		}
			l.Error("get_order_error", "status", 500, "reason", "internal error", "error", err)
			return echo.NewHTTPError(http.StatusForbidden, "internal error")
	}

	l.Info("get_order_success")
	return c.JSON(http.StatusOK, order)
}

func(h *OrderHTTP) UpdateOrder(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "order.update_order")

	idstr := c.Param("id")
	id, err := uuid.Parse(idstr)
	if err != nil {
		l.Error("update_order_error", "status", 400, "reason", "id is not uuid", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "id is not uuid")
	}

	var req struct {
		Status models.OrderStatus `json:"status"`
	}
	if err := c.Bind(&req); err != nil {
		l.Error("update_order_error", "status", 400, "reason", "status is not alowded", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "status is not alowded")
	}

	order, err := h.Svc.UpdateOrder(ctx, id, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			l.Error("update_order_error", "status", 404, "reason", "record not found", "error", err)
			return echo.NewHTTPError(http.StatusNotFound, "record not found")
		}
		l.Error("update_order_error", "status", 500, "reason", "internal error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}

	l.Info("update_order_success")
	return c.JSON(http.StatusOK, order)
}

func (h *OrderHTTP) CancelOrder(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "order.cancel_order")

	idstr := c.Param("id")
	id, err := uuid.Parse(idstr)
	if err != nil {
		l.Error("cancel_order_error", "status", 400, "reason", "id is not uuid", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "id is not uuid")
	}

	userID, err := h.GetID(c)
	if err != nil {
		l.Warn("cancel_orders_error", "status", 401, "reason", "unauthorized", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}	

	order, err := h.Svc.CancelOrder(ctx, id, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			l.Error("cancel_order_error", "status", 404, "reason", "record not found", "error", err)
			return echo.NewHTTPError(http.StatusNotFound, "record not found")
		}
		l.Error("cancel_order_error", "status", 500, "reason", "internal error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}

	l.Info("cancel_order_success")
	return c.JSON(http.StatusOK, order)
}