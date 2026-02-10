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