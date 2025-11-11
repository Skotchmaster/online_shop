package repo

import (
	"errors"
	"time"

	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/pkg/tokens"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func(r *GormRepo) AddRefreshToDB(refreshToken string) error {
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
		Token: jwthelp.Sha256Hex(refreshToken),
		UserID: userID,
		ExpiresAt: claims.ExpiresAt.Time.Unix(),
		JTI: claims.ID,
	}

	if err := r.DB.Create(&refreshModel).Error; err != nil {
		return err
	}

	return nil
}

func (r *GormRepo) FindRefreshByID(tokenID string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.DB.Where("jti = ?", tokenID).First(&token).Error; err != nil{
		return nil, err
	}
	return &token, nil
}

func (r *GormRepo) RefreshExists(tokenID string) (bool, error) {
    var count int64
    if err := r.DB.Model(&models.RefreshToken{}).
        Where("jti = ?", tokenID).
        Count(&count).Error; err != nil {
        return false, err
    }
    return count > 0, nil
}

func (r *GormRepo) refreshExpiredOrRevoked(db *gorm.DB, tokenID string) (bool, error) {
    var refresh models.RefreshToken
    if err := db.Where("jti = ?", tokenID).First(&refresh).Error; err != nil {
        return false, err
    }
    if refresh.ExpiresAt < time.Now().Unix() || refresh.Revoked {
        return true, nil
    }
    return false, nil
}

func (r *GormRepo) markAsUsed(db *gorm.DB, tokenID string) error {
    return db.Model(&models.RefreshToken{}).
        Where("jti = ?", tokenID).
        Update("revoked", true).Error
}

func (r *GormRepo) RefreshExpiredOrRevoked(tokenID string) (bool, error) {
    return r.refreshExpiredOrRevoked(r.DB, tokenID)
}

func (r *GormRepo) MarkAsUsed(tokenID string) error {
    return r.markAsUsed(r.DB, tokenID)
}

func (r *GormRepo) RotateRefreshToken(oldJTI string, newToken models.RefreshToken) error {
    return r.DB.Transaction(func(tx *gorm.DB) error {
        expired, err := r.refreshExpiredOrRevoked(tx, oldJTI)
        if err != nil {
            return err
        }
        if expired {
            return errors.New("token expired or revoked")
        }

        if err := r.markAsUsed(tx, oldJTI); err != nil {
            return err
        }

        if err := tx.Create(&newToken).Error; err != nil {
            return err
        }

        return nil
    })
}