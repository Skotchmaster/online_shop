package service

import (
	"context"
	"errors"

	"github.com/Skotchmaster/online_shop/services/catalog/internal/models"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/repo"
	"github.com/google/uuid"
)

type CatalogService struct {
	Repo *repo.GormRepo
}

func (s *CatalogService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	return s.Repo.GetProduct(ctx, id)
}

func (s *CatalogService) GetProducts(ctx context.Context, offset, limit int) (int64, *[]models.Product, string, error) {
	return s.Repo.GetProducts(ctx, offset, limit)
}

func(s *CatalogService) CreateProduct(ctx context.Context, product *models.Product) error {
	return s.Repo.CreateProduct(ctx, product)
}

func(s *CatalogService) PatchProduct(ctx context.Context, req models.Product, id uuid.UUID) (*models.Product, error) {
	if req.Price < 0 {
        return nil, errors.New("price cannot be negative")
    }
    
    return s.Repo.PatchProduct(ctx, req, id)
}

func(s *CatalogService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	return s.Repo.DeleteProduct(ctx, id)
}

