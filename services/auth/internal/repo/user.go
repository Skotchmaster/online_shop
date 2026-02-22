package repo

import (
	"context"
	"errors"

	pkg_hash "github.com/Skotchmaster/online_shop/pkg/hash"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUserAlreadyExist = errors.New("user already exist")

func (r *GormRepo) UserExist(ctx context.Context, username, password string) (*models.User, error) {
	var user models.User
	if err := r.DB.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !pkg_hash.CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

func (r *GormRepo) CreateUserIfNotExists(ctx context.Context, u *models.User) error {
    tx := r.DB.WithContext(ctx).Where("username = ?", u.Username).FirstOrCreate(u)
    if tx.Error != nil {
        return tx.Error
    }
    if tx.RowsAffected == 0 {
        return ErrUserAlreadyExist
    }
    return nil
}


func (r *GormRepo) LogOut(ctx context.Context, refreshtoken string) error {
	result := r.DB.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("token = ?", jwthelp.Sha256Hex(refreshtoken)).
		Update("revoked", true)
	return result.Error
}

func (r *GormRepo) GetUserById(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.DB.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}