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

	return items, err
}

func (h *CartService) AddToCart(ctx context.Context, item *models.CartItem) error {
	err := h.Repo.AddToCart(ctx, item)

	return err
}

func (h *CartService) DeleteOneFromCart(ctx context.Context, productID uuid.UUID, userID uuid.UUID) (bool, models.CartItem, error) {
	deleted, item, err := h.Repo.DeleteOneFromCart(ctx, productID, userID)
	
	return deleted, item, err
}

func (h *CartService) DeleteAllFromCart(ctx context.Context, userID uuid.UUID) error {
	err := h.Repo.DeleteAllFromCart(ctx, userID)

	return err
}