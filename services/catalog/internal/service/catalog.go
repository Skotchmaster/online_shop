package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Skotchmaster/online_shop/services/catalog/internal/models"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/repo"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/transport"
	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation")

type CatalogService struct {
	Repo *repo.GormRepo
}

func (s *CatalogService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	return s.Repo.GetProduct(ctx, id)
}

func (s *CatalogService) GetProducts(ctx context.Context, offset, limit int) (int64, *[]models.Product, string, error) {
	return s.Repo.GetProducts(ctx, offset, limit)
}

func (s *CatalogService) CreateProduct(ctx context.Context, req transport.CreateProductRequest) (*models.Product, error) {

	prod := models.Product{
        Name: req.Name,
        Description: req.Description,
        Price: req.Price,
        Count: req.Count,
    }

    if req.Name == "" || req.Description == "" {
        return nil, fmt.Errorf("%w: name and description are required", ErrValidation)
    }
    if req.Price < 0 {
        return nil, fmt.Errorf("%w: price must be >= 0", ErrValidation)
    }

    return s.Repo.CreateProduct(ctx, &prod)
}

func (s *CatalogService) PatchProduct(ctx context.Context, req transport.PatchProductRequest, id uuid.UUID) (*models.Product, error) {

    if req.Price != nil && *req.Price < 0 {
		return nil, fmt.Errorf("%w: price must be >= 0", ErrValidation)
	}
	if req.Name != nil && *req.Name == "" {
		return nil, fmt.Errorf("%w: name cannot be empty", ErrValidation)
	}
	if req.Name == nil && req.Description == nil && req.Price == nil && req.Count == nil {
		return nil, fmt.Errorf("%w: empty patch", ErrValidation)
	}

    return s.Repo.PatchProduct(ctx, req, id)
}


func(s *CatalogService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	return s.Repo.DeleteProduct(ctx, id)
}

func (s *CatalogService) SearchProducts(ctx context.Context, rawQ string, offset, limit int) (int64, *[]models.Product, string, error) {
	q := strings.TrimSpace(rawQ)
	if q == "" {
		empty := []models.Product{}
		return 0, &empty, "", nil
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.Repo.SearchProducts(ctx, q, offset, limit)
}
