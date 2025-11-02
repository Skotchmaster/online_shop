package models

type OrderItem struct {
	ID        uint `gorm:"primaryKey"                  json:"id"`
	OrderID   uint `gorm:"not null"                    json:"order_id"`
	UserID    uint `gorm:"index;not null"              json:"user_id"`
	ProductID uint `gorm:"not null"                    json:"product_id"`
	Quantity  uint `gorm:"default:1;check:quantity>0"  json:"quantity"`
}

type Order struct {
	ID        uint    `gorm:"primaryKey"                  json:"id"`
	UserID    uint    `gorm:"index;not null"              json:"user_id"`
	CreatedAt int64   `gorm:"not null"                    json:"created_at"`
	Total     float64 `gorm:"not null"                    json:"total"`
	Status    string  `gorm:"not null"                    json:"status"`
}