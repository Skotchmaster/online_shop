package repo

import (
	"context"

	"github.com/Skotchmaster/online_shop/services/catalog/internal/models"
	"github.com/google/uuid"
)

func (r *GormRepo) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	product := models.Product{}
	if err := r.DB.WithContext(ctx).Where("ID=?", id).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func(r *GormRepo) GetProducts(ctx context.Context, offset, limit int) (int64, *[]models.Product, string, error) {
	var total int64
	if err := r.DB.WithContext(ctx).Model(models.Product{}).Count(&total).Error; err != nil{
		return 0, nil, "cannot count total", err
	}

	var items []models.Product
	if err := r.DB.WithContext(ctx).Model(&models.Product{}).Order("id ASC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return 0, nil, "cannot get products with offset", err
	}

	return total, &items, "", nil
}

func(r *GormRepo) CreateProduct(ctx context.Context, prod *models.Product) error {
	if err := r.DB.WithContext(ctx).Create(&prod).Error; err != nil {
		return err
	}
	return nil
}

func(r *GormRepo) PatchProduct(ctx context.Context, req models.Product, id uuid.UUID) (*models.Product, error) {
	var prod models.Product
	if err := r.DB.WithContext(ctx).First(&prod, id).Error; err != nil {
		return nil, err
	}

	prod.Name = req.Name
	prod.Description = req.Description
	prod.Price = req.Price
	prod.Count = req.Count

	if err := r.DB.WithContext(ctx).Save(&prod).Error; err != nil {
		return nil, err
	}

	return &prod, nil
}

func(r *GormRepo) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	if err := r.DB.WithContext(ctx).Delete(&models.Product{}, id).Error; err != nil {
		return err
	}
	return nil
}