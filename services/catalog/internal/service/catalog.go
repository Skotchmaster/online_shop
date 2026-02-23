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
	"gorm.io/gorm"
)

var(
	ErrValidation = errors.New("validation")
	ErrNotFound = errors.New("not found")
) 

type CatalogService struct {
	Repo *repo.GormRepo
}

func (s *CatalogService) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	item, err := s.Repo.GetProduct(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("item not found: %w", ErrNotFound)
		} else {
			return nil, err
		}
	}
	return item, nil
}

func (s *CatalogService) GetProducts(ctx context.Context, offset, limit int) (int64, *[]models.Product, error) {
	return s.Repo.GetProducts(ctx, offset, limit)
}

func (s *CatalogService) CreateProduct(ctx context.Context, req transport.CreateProductRequest) (*models.Product, error) {
    if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Description) == "" {
        return nil, fmt.Errorf("name and description are required: %w", ErrValidation)
    }
    if req.Price < 0 {
        return nil, fmt.Errorf("price must be >= 0: %w", ErrValidation)
    }

	prod := models.Product{
        Name: req.Name,
        Description: req.Description,
        Price: req.Price,
        Count: req.Count,
    }

    return s.Repo.CreateProduct(ctx, &prod)
}

func (s *CatalogService) PatchProduct(ctx context.Context, req transport.PatchProductRequest, id uuid.UUID) (*models.Product, error) {

    if req.Price != nil && *req.Price < 0 {
		return nil, fmt.Errorf("price must be >= 0: %w", ErrValidation)
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, fmt.Errorf("name cannot be empty: %w", ErrValidation)
	}
	if req.Description != nil && strings.TrimSpace(*req.Description) == "" {
		return nil, fmt.Errorf("description cannot be empty: %w", ErrValidation)
	}
	if req.Name == nil && req.Description == nil && req.Price == nil && req.Count == nil {
		return nil, fmt.Errorf("empty patch: %w", ErrValidation)
	}

    item, err := s.Repo.PatchProduct(ctx, req, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return item, err
}


func(s *CatalogService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	err := s.Repo.DeleteProduct(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *CatalogService) SearchProducts(ctx context.Context, rawQ string, offset, limit int) (int64, *[]models.Product, error) {
	q := strings.TrimSpace(rawQ)
	if q == "" {
		return 0, &[]models.Product{},  nil
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
