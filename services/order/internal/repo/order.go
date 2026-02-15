package repo

import (
	"context"

	"github.com/Skotchmaster/online_shop/services/order/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormRepo struct {
	DB *gorm.DB
}

func(r *GormRepo) CreateOrder(ctx context.Context, order *models.Order) (*models.Order, error) {
	if err := r.DB.WithContext(ctx).Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

func (r *GormRepo) ListOrders(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Order, error) {
	q := r.DB.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID)

	var orders []models.Order
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&orders).Error; err != nil {
		return nil, err
	}	
	return orders, nil
}

func(r *GormRepo) GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	var order models.Order
	if err := r.DB.WithContext(ctx).Preload("Items").Where("ID = ?", id).First(&order).Error; err != nil{
		return nil, err
	}
	return &order, nil
}

func(r *GormRepo) UpdateOrder(ctx context.Context, id uuid.UUID, prev models.OrderStatus, curr models.OrderStatus) (*models.Order, error) {
	res := r.DB.WithContext(ctx).Where("id = ? AND status = ?", id, prev).Update("status", curr)

	if res.RowsAffected == 0 {

		ord, err := r.GetOrder(ctx, id)

		if ord.Status == curr {
			return ord, nil
		}

		return nil, err
	}

	return r.GetOrder(ctx, id)
}

func(r *GormRepo) CancelOrder(ctx context.Context, id uuid.UUID, status models.OrderStatus) (*models.Order, error) {

	res := r.DB.WithContext(ctx).Where("id = ? AND status = ", id, status).Update("status", models.OrderStatusCancelled)

	if res.RowsAffected == 0 {

		ord, err := r.GetOrder(ctx, id)

		if ord.Status == status {
			return ord, nil
		}

		return nil, err
	}

	return r.GetOrder(ctx, id)
}