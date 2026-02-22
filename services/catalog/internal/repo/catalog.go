package repo

import (
	"context"

	"github.com/Skotchmaster/online_shop/services/catalog/internal/models"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/transport"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *GormRepo) GetProduct(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	product := models.Product{}
	if err := r.DB.WithContext(ctx).Where("ID=?", id).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func(r *GormRepo) GetProducts(ctx context.Context, offset, limit int) (int64, *[]models.Product, error) {
	var total int64
	if err := r.DB.WithContext(ctx).Model(models.Product{}).Count(&total).Error; err != nil{
		return 0, nil, err
	}

	var items []models.Product
	if err := r.DB.WithContext(ctx).Model(&models.Product{}).Order("id ASC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, &items, nil
}

func(r *GormRepo) CreateProduct(ctx context.Context, prod *models.Product) (*models.Product, error) {
	if err := r.DB.WithContext(ctx).Create(prod).Error; err != nil {
		return nil, err
	}
	return prod, nil
}

func(r *GormRepo) PatchProduct(ctx context.Context, req transport.PatchProductRequest, id uuid.UUID) (*models.Product, error) {
	var prod models.Product
	if err := r.DB.WithContext(ctx).First(&prod, id).Error; err != nil {
		return nil, err
	}

	if req.Name != nil {
		prod.Name = *req.Name
	}
	if req.Description != nil {
		prod.Description = *req.Description
	}
	if req.Price != nil {
		prod.Price = *req.Price
	}
	if req.Count != nil {
		prod.Count = *req.Count
	}

	if err := r.DB.WithContext(ctx).Save(&prod).Error; err != nil {
		return nil, err
	}

	return &prod, nil
}

func(r *GormRepo) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	res := r.DB.WithContext(ctx).Delete(&models.Product{}, id)
	
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil

}

func (r *GormRepo) SearchProducts(ctx context.Context, q string, offset, limit int) (int64, *[]models.Product, error) {
	ftsQuerySQL := `(websearch_to_tsquery('russian', unaccent(?)) || websearch_to_tsquery('english', unaccent(?)))`
	ftsWhere := "search_vector @@ " + ftsQuerySQL

	var totalFTS int64
	if err := r.DB.WithContext(ctx).
		Model(&models.Product{}).
		Where(ftsWhere, q, q).
		Count(&totalFTS).Error; err != nil {
		return 0, nil, err
	}

	if totalFTS > 0 {
		items := make([]models.Product, 0, limit)
		orderExpr := clause.Expr{
			SQL:  "ts_rank_cd(search_vector, "+ftsQuerySQL+") DESC",
			Vars: []any{q, q},
		}

		if err := r.DB.WithContext(ctx).
			Model(&models.Product{}).
			Where(ftsWhere, q, q).
			Order(orderExpr).
			Limit(limit).
			Offset(offset).
			Find(&items).Error; err != nil {
			return 0, nil, err
		}

		return totalFTS, &items, nil
	}

	var totalTrgm int64
	if err := r.DB.WithContext(ctx).
		Model(&models.Product{}).
		Where("name % ? OR description % ?", q, q).
		Count(&totalTrgm).Error; err != nil {
		return 0, nil, err
	}

	items := make([]models.Product, 0, limit)
	if err := r.DB.WithContext(ctx).
		Model(&models.Product{}).
		Where("name % ? OR description % ?", q, q).
		Order(clause.Expr{
			SQL:  "GREATEST(similarity(name, ?), similarity(description, ?)) DESC",
			Vars: []any{q, q},
		}).
		Limit(limit).
		Offset(offset).
		Find(&items).Error; err != nil {
		return 0, nil, err
	}

	return totalTrgm, &items, nil
}