package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartItem struct {
	ID        uuid.UUID `gorm:"primaryKey"                              json:"id"`
	UserID    uuid.UUID `gorm:"uniqueIndex:idx_user_product;not null"  json:"user_id"`
	ProductID uuid.UUID `gorm:"uniqueIndex:idx_user_product;not null"   json:"product_id"`
	Quantity  uint      `gorm:"default:1;check:quantity>0"              json:"quantity"`
}


func (c *CartItem) BeforeCreate(tx *gorm.DB) error {
    if c.ID == uuid.Nil {
        c.ID = uuid.New()
    }
    return nil
}

func (CartItem) TableName() string {
    return "cart_items"
}