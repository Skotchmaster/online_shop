package transport

import "github.com/google/uuid"

type DeleteOneFromCartResponse struct {
	ProductID uuid.UUID `json:"product_id"`
	Deleted   bool      `json:"deleted"`
	Quantity  uint      `json:"quantity"`
}
