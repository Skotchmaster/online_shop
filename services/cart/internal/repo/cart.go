package repo

import (
	"context"

	"github.com/Skotchmaster/online_shop/services/cart/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"github.com/google/uuid"
)

func (r *GormRepo) GetCart(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error) {
	var items []models.CartItem
	if err := r.DB.WithContext(ctx).Where("user_id=?", userID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *GormRepo) AddToCart(ctx context.Context, item *models.CartItem) error {
    return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        res := tx.Model(&models.CartItem{}).
            Where("user_id = ? AND product_id = ?", item.UserID, item.ProductID).
            Update("quantity", gorm.Expr("quantity + ?", item.Quantity))
        if res.Error != nil {
            return res.Error
        }
        if res.RowsAffected > 0 {
            return tx.Where("user_id = ? AND product_id = ?", item.UserID, item.ProductID).First(item).Error
        }

        if err := tx.Create(item).Error; err != nil {
            return err
        }
        return nil
    })
}

func (r *GormRepo) DeleteOneFromCart(ctx context.Context, productID uuid.UUID, userID uuid.UUID) (bool, *models.CartItem, error) {
	var item models.CartItem
	deleted := false

	if err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("product_id = ? AND user_id = ?", productID, userID).First(&item).Error; err != nil {
			return err
		}
		if item.Quantity > 1 {
			if err := tx.Model(&item).Update("quantity", gorm.Expr("quantity - 1")).Error; err != nil {
				return err
			}
            if err := tx.Where("product_id = ? AND user_id = ?", productID, userID).First(&item).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Delete(&item).Error; err != nil {
				return err
			}
			deleted = true
		}
		return nil
	}); err != nil{
		return  false, nil, err
	}
	return deleted, &item, nil
}

func (r *GormRepo) DeleteAllFromCart(ctx context.Context, userID uuid.UUID) error {
    return r.DB.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.CartItem{}).Error
}