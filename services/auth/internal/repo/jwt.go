package repo

import (
	"time"

	"github.com/Skotchmaster/online_shop/pkg/tokens"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"github.com/google/uuid"
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

func (r *GormRepo) MarkAsUsed(tokenID string) error {
	if err := r.DB.Update("revoked", true).Where("jti= ?", tokenID).Error; err != nil{
		return err
	}
	return nil
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

func (r *GormRepo) RefreshExpiredOrRevoked(tokenID string) (bool, error) {
	var refresh models.RefreshToken
	if err := r.DB.Where("jti = ?", tokenID).First(&refresh).Error; err != nil{
		return false, err
	}
	if refresh.ExpiresAt < time.Now().Unix() || !refresh.Revoked {
		return false, nil
	}
	return true, nil
}