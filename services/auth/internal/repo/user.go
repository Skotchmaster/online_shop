package repo

import (
	"errors"

	pkg_hash "github.com/Skotchmaster/online_shop/pkg/hash"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUserAlreadyExist = errors.New("user already exist")

func (r *GormRepo) UserExist(username, password string) (*models.User, error) {
	var user models.User
	if err := r.DB.Where("username = ?", username).First(&user).Error; err != nil {
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

func (r *GormRepo) UserNotExist(user models.User) (error) {
	if err := r.DB.Where("username = ?", user.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			if res := r.DB.Create(&user); res != nil {
				return err
			} else {
				return nil
			}
		}
		return err
	}
	return ErrUserAlreadyExist
}

func (r *GormRepo) LogOut(refreshtoken string) error {
	result := r.DB.Model(&models.RefreshToken{}).
		Where("token = ?", jwthelp.Sha256Hex(refreshtoken)).
		Update("revoked", true)
	return result.Error
}