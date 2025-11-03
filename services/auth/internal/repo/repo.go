package repo

import (
	"github.com/Skotchmaster/online_shop/pkg/tokens"
	jwthelp "github.com/Skotchmaster/online_shop/services/auth/internal/jwt"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormRepo struct{ 
	DB *gorm.DB 
	JWTSecret []byte
	RefreshSecret []byte
}

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