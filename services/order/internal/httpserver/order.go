package httpserver

import (
	"errors"
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/logging"
	"github.com/Skotchmaster/online_shop/services/order/internal/service"
	"github.com/Skotchmaster/online_shop/services/order/internal/transport"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type OrderHTTP struct {
	Svc *service.OrserService
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
		l.Warn("create_order_error", "status", 400, "reason", "invalid body", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
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
			return echo.NewHTTPError(http.StatusBadRequest, "internal error")
		}
	}

	l.Info("create_order_success")
	return c.JSON(http.StatusCreated, order)
}