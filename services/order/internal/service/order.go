package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Skotchmaster/online_shop/services/order/internal/models"
	"github.com/Skotchmaster/online_shop/services/order/internal/repo"
	"github.com/Skotchmaster/online_shop/services/order/internal/transport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrValidation = errors.New("validation") // 400
	ErrForbidden  = errors.New("validation") // 403
	ErrNotFound   = errors.New("not found")  // 404
	ErrConflict   = errors.New("conflict")   // 409
)

func canTransition(from, to models.OrderStatus) bool {
	allowed := map[models.OrderStatus]map[models.OrderStatus]bool{
		models.OrderStatusNew: {
			models.OrderStatusPaid:      true,
			models.OrderStatusCancelled: true,
		},
		models.OrderStatusPaid: {
			models.OrderStatusShipped:   true,
			models.OrderStatusCancelled: true,
		},
		models.OrderStatusShipped: {
			models.OrderStatusDone: true,
		},
		models.OrderStatusDone:      {},
		models.OrderStatusCancelled: {},
	}
	return allowed[from][to]
}

type OrderService struct {
	Repo *repo.GormRepo
}

func (svc *OrderService) CreateOrder(ctx context.Context, req transport.CreateOrderRequest, userID uuid.UUID) (*models.Order, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: items required", ErrValidation)
	}

	var total int64
	var items []models.OrderItem

	for i := range req.Items {
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
			Quantity:  req.Items[i].Quantity,
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

	return svc.Repo.CreateOrder(ctx, order)
}

func (svc *OrderService) ListOrders(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error) {
	return svc.Repo.ListOrders(ctx, userID, limit, offset)
}

func (svc *OrderService) GetOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error) {
	order, err := svc.Repo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrNotFound
	}
	return order, nil
}

func (svc *OrderService) UpdateOrder(ctx context.Context, id uuid.UUID, status models.OrderStatus) (*models.Order, error) {
	order, err := svc.Repo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	prev := order.Status

	if !canTransition(prev, status) {
		return nil, ErrConflict
	}

	updated, err := svc.Repo.UpdateOrder(ctx, id, prev, status)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

func (svc *OrderService) CancelOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error) {
	order, err := svc.Repo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.UserID != userID {
		return nil, ErrForbidden
	}

	status := order.Status

	if status != models.OrderStatusNew {
		return nil, ErrConflict
	}

	updated, err := svc.Repo.CancelOrder(ctx, id, models.OrderStatusNew)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return updated, err
}
