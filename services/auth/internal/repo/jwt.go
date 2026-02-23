package repo

import (
	"context"
	"errors"
	"time"

	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/pkg/tokens"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrTokenExpiredOrRevoked = errors.New("token expired or revoked")

func (r *GormRepo) AddRefreshToDB(ctx context.Context, refreshToken string) error {
	claims, err := tokens.RefreshClaimsFromToken(refreshToken, r.RefreshSecret)
	if err != nil {
		return err
	}
	sub := claims.Subject
	userID, err := uuid.Parse(sub)
	if err != nil {
		return err
	}
	refreshModel := models.RefreshToken{
		Token:     jwthelp.Sha256Hex(refreshToken),
		UserID:    userID,
		ExpiresAt: claims.ExpiresAt.Time,
		JTI:       claims.ID,
	}

	if err := r.DB.WithContext(ctx).Create(&refreshModel).Error; err != nil {
		return err
	}

	return nil
}

func (r *GormRepo) FindRefreshByID(ctx context.Context, tokenID string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.DB.WithContext(ctx).Where("jti = ?", tokenID).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *GormRepo) RefreshExists(ctx context.Context, tokenID string) (bool, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("jti = ?", tokenID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GormRepo) refreshExpiredOrRevoked(ctx context.Context, db *gorm.DB, tokenID string) (bool, error) {
	var refresh models.RefreshToken
	if err := db.WithContext(ctx).Where("jti = ?", tokenID).First(&refresh).Error; err != nil {
		return false, err
	}
	if refresh.ExpiresAt.Before(time.Now()) || refresh.Revoked {
		return true, nil
	}
	return false, nil
}

func (r *GormRepo) markAsUsed(ctx context.Context, db *gorm.DB, tokenID string) error {
	return db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("jti = ?", tokenID).
		Update("revoked", true).Error
}

func (r *GormRepo) RefreshExpiredOrRevoked(ctx context.Context, tokenID string) (bool, error) {
	return r.refreshExpiredOrRevoked(ctx, r.DB, tokenID)
}

func (r *GormRepo) MarkAsUsed(ctx context.Context, tokenID string) error {
	return r.markAsUsed(ctx, r.DB, tokenID)
}

func (r *GormRepo) RotateRefreshToken(ctx context.Context, oldJTI string, newToken models.RefreshToken) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		expired, err := r.refreshExpiredOrRevoked(ctx, tx, oldJTI)
		if err != nil {
			return err
		}
		if expired {
			return ErrTokenExpiredOrRevoked
		}

		if err := r.markAsUsed(ctx, tx, oldJTI); err != nil {
			return err
		}

		if err := tx.Create(&newToken).Error; err != nil {
			return err
		}

		return nil
	})
}
