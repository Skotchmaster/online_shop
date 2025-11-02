package models

type CartItem struct {
	ID        uint `gorm:"primaryKey"                  json:"id"`
	UserID    uint `gorm:"index;not null"              json:"user_id"`
	ProductID uint `gorm:"not null"                    json:"product_id"`
	Quantity  uint `gorm:"default:1;check:quantity>0"  json:"quantity"`
}