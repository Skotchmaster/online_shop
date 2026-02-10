package repo

import (
	"context"

	"github.com/Skotchmaster/online_shop/services/order/internal/models"
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