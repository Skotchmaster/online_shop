package transport

import "github.com/google/uuid"

type CreateOrderItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	UnitPrice int64     `json:"unit_price"`
}

type CreateOrderRequest struct {
	Items []CreateOrderItem `json:"items"`
}