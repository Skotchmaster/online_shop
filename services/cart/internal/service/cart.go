package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Skotchmaster/online_shop/services/cart/internal/models"
	"github.com/Skotchmaster/online_shop/services/cart/internal/repo"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrValidation = errors.New("validation")
	ErrNotFound = errors.New("not found")
)

type CartService struct {
	Repo        *repo.GormRepo
}

func (h *CartService) GetCart(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error) {
	items, err := h.Repo.GetCart(ctx, userID)

	return items, err
}

func (h *CartService) AddToCart(ctx context.Context, item *models.CartItem) error {
	if item.ProductID == uuid.Nil{
		return fmt.Errorf("ID product must be not nil: %w", ErrValidation)
	}
	if item.Quantity == 0 {
		return fmt.Errorf("quantity must be more than zero: %w", ErrValidation)
	}

	err := h.Repo.AddToCart(ctx, item)

	return err
}

func (h *CartService) DeleteOneFromCart(ctx context.Context, productID uuid.UUID, userID uuid.UUID) (bool, *models.CartItem, error) {
	if productID == uuid.Nil {
		return false, nil, fmt.Errorf("ID product must be not nil: %w", ErrValidation)
	}

	deleted, item, err := h.Repo.DeleteOneFromCart(ctx, productID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, fmt.Errorf("product not found: %w", ErrNotFound)
	} 

	return deleted, item, err
}

func (h *CartService) DeleteAllFromCart(ctx context.Context, userID uuid.UUID) error {
	err := h.Repo.DeleteAllFromCart(ctx, userID)

	return err
}