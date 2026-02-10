package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "NEW"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusDone      OrderStatus = "DONE"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

type Order struct {
	ID        uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID   `gorm:"type:uuid;not null;index" json:"user_id"`
	Status    OrderStatus `gorm:"type:text;not null" json:"status"`
	Total     int64       `gorm:"type:bigint;not null" json:"total"`
	CreatedAt time.Time   `gorm:"type:timestamptz;not null" json:"created_at"`
	UpdatedAt time.Time   `gorm:"type:timestamptz;not null" json:"updated_at"`

	Items []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.Status == "" {
		o.Status = OrderStatusNew
	}
	return nil
}

type OrderItem struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`

	Quantity  int   `gorm:"not null;check:quantity > 0" json:"quantity"`
	UnitPrice int64 `gorm:"type:bigint;not null;check:unit_price >= 0" json:"unit_price"`
	LineTotal int64 `gorm:"type:bigint;not null;check:line_total >= 0" json:"line_total"`
}

func (i *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}