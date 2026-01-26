package models

import "github.com/google/uuid"

type CartItem struct {
	ID        uuid.UUID `gorm:"primaryKey"                              json:"id"`
	UserID    uuid.UUID `gorm:"uniqueIndex:idx_user_product;not nulll"  json:"user_id"`
	ProductID uuid.UUID `gorm:"uniqueIndex:idx_user_product;not null"   json:"product_id"`
	Quantity  uint      `gorm:"default:1;check:quantity>0"              json:"quantity"`
}
