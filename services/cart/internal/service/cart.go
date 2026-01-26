package service

import (
	"context"

	"github.com/Skotchmaster/online_shop/services/cart/internal/models"
	"github.com/Skotchmaster/online_shop/services/cart/internal/repo"
	"github.com/google/uuid"
)

type CartService struct {
	Repo        *repo.GormRepo
}

func (h *CartService) GetCart(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error) {
	items, err := h.Repo.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (h *CartService) AddToCart(ctx context.Context, item *models.CartItem) error {
	if err := h.Repo.AddToCart(ctx, item); err != nil {
		return err
	}

	return nil
}

func (h *CartService) DeleteOneFromCart(ctx context.Context, productID uuid.UUID, userID uuid.UUID) (bool, models.CartItem, error) {
	deleted, item, err := h.Repo.DeleteOneFromCart(ctx, productID, userID)
	if err != nil {
		return false, models.CartItem{}, err
	}
	
	return deleted, item, err
}

func (h *CartService) DeleteAllFromCart(ctx context.Context, userID uuid.UUID) error {
	err := h.Repo.DeleteAllFromCart(ctx, userID)
	if err != nil {
		return err
	}

	return nil
}

