package search

import (
	"context"
	"strings"
	"time"

	"github.com/Skotchmaster/online_shop/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Results struct {
	Total int64
	Items []models.Product
}

func sanitizeQuery(q string) string {
	return strings.TrimSpace(q)
}

func ftsQuerySQL() string {
	return `
(
  websearch_to_tsquery('russian', unaccent(?)) ||
  websearch_to_tsquery('english', unaccent(?))
)
`
}

func Search(db *gorm.DB, rawQ string, offset, limit int) (Results, error) {
	q := sanitizeQuery(rawQ)
	if q == "" {
		return Results{Total: 0, Items: []models.Product{}}, nil
	}
	if limit <= 0 { limit = 20 }
	if limit > 100 { limit = 100 }
	if offset < 0 { offset = 0 }

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tx := db.WithContext(ctx)

	ftsWhere := "search_vector @@ " + ftsQuerySQL()

	var totalFTS int64
	if err := tx.Model(&models.Product{}).
		Where(ftsWhere, q, q).
		Count(&totalFTS).Error; err != nil {
		return Results{}, err
	}

	if totalFTS > 0 {
		items := make([]models.Product, 0, limit)

		orderExpr := clause.Expr{
			SQL:  "ts_rank_cd(search_vector, "+ftsQuerySQL()+") DESC",
			Vars: []interface{}{q, q},
		}

		if err := tx.
			Where(ftsWhere, q, q).
			Order(orderExpr).
			Limit(limit).
			Offset(offset).
			Find(&items).Error; err != nil {
			return Results{}, err
		}

		return Results{Total: totalFTS, Items: items}, nil
	}

	var totalTrgm int64
	if err := tx.Model(&models.Product{}).
		Where("name % ? OR description % ?", q, q).
		Count(&totalTrgm).Error; err != nil {
		return Results{}, err
	}

	items := make([]models.Product, 0, limit)
	if err := tx.
		Where("name % ? OR description % ?", q, q).
		Order(clause.Expr{
			SQL:  "GREATEST(similarity(name, ?), similarity(description, ?)) DESC",
			Vars: []interface{}{q, q},
		}).
		Limit(limit).
		Offset(offset).
		Find(&items).Error; err != nil {
		return Results{}, err
	}

	return Results{Total: totalTrgm, Items: items}, nil
}
