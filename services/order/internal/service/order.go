package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Skotchmaster/online_shop/services/order/internal/models"
	"github.com/Skotchmaster/online_shop/services/order/internal/repo"
	"github.com/Skotchmaster/online_shop/services/order/internal/transport"
	"github.com/google/uuid"
)

var (
	ErrValidation = errors.New("validation") // 400
	ErrNotFound   = errors.New("not found")  // 404
	ErrConflict   = errors.New("conflict")   // 409
)

type OrserService struct {
	repo *repo.GormRepo
}

func(svc *OrserService) CreateOrder(ctx context.Context, req transport.CreateOrderRequest, userID uuid.UUID) (*models.Order, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: items required", ErrValidation)
	}

	var total int64
	var items []models.OrderItem

	for i := range(req.Items) {
		if req.Items[i].ProductID == uuid.Nil {
			return nil, fmt.Errorf("%w: product_id required", ErrValidation)
		}
		if req.Items[i].Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity must be > 0", ErrValidation)
		}
		if req.Items[i].UnitPrice < 0 {
			return nil, fmt.Errorf("%w: price must be >= 0", ErrValidation)
		}

		lineTotal := int64(req.Items[i].Quantity) * req.Items[i].UnitPrice

		item := models.OrderItem{
			ProductID: req.Items[i].ProductID,
			Quantity: req.Items[i].Quantity,
			UnitPrice: req.Items[i].UnitPrice,
			LineTotal: lineTotal,
		}

		total += lineTotal
		items = append(items, item)
	}

	order := &models.Order{
		UserID: userID,
		Status: models.OrderStatusNew,
		Total:  total,
		Items:  items,
	}

	return svc.repo.CreateOrder(ctx, order)
}