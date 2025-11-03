package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	pkg_hash "github.com/Skotchmaster/online_shop/pkg/hash"
	"github.com/Skotchmaster/online_shop/pkg/logging"
	"github.com/Skotchmaster/online_shop/pkg/tokens"
	jwthelp "github.com/Skotchmaster/online_shop/services/auth/internal/jwt"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"github.com/Skotchmaster/online_shop/services/auth/internal/repo"
)

type AuthService struct {
	Repo     repo.GormRepo
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	AccessExp    time.Time
	RefreshExp   time.Time
	IsAdmin      bool
}

func (h *AuthService) Register(ctx context.Context, username, password string) error {
	l := logging.FromContext(ctx).With("svc", "auth.register")

	pwHash, err := pkg_hash.HashPassword(password)
	if err != nil {
		l.Error("register_error", "status", 500, "reason", "cannot hash the password", "error", err)
		return err
	}
	user := models.User{
		Username:     username,
		PasswordHash: string(pwHash),
		Role:         "user"}
	var userCheck models.User
	if err := h.Repo.DB.Where("username = ?", username).First(&userCheck).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			l.Error("register_error", "status", 500, "reason", "db_error", "error", err)
			return err
		}
	} else {
		l.Warn("register_failed", "status", 409, "reason", "user_exists")
		return err
	}
	if err := h.Repo.DB.Create(&user).Error; err != nil {
		l.Error("register_failed", "status", 500, "reason", "db_error", "error", err)
		return err
	}

	return nil
}

func (h *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	l := logging.FromContext(ctx).With("svc", "auth.login", "username", username)
	var user models.User
	if err := h.Repo.DB.Where("username = ?", username).First(&user).Error; err != nil {
		l.Warn("user_lookup_failed", "error", err)
		return nil, err
	}
	if !pkg_hash.CheckPassword(user.PasswordHash, password) {
		l.Warn("mismatch password")
		return nil, errors.New("mismatch password")
	}

	accessExp := time.Now().Add(time.Minute * 15)
	accessClaims := tokens.AccessClaims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}

	tokenAccess := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := tokenAccess.SignedString(h.Repo.JWTSecret)
	if err != nil {
		return nil, err
	}

	jti := jwthelp.NewJTI()
	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := tokens.RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			ID:        jti,
		},
	}

	tokenRefresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := tokenRefresh.SignedString(h.Repo.RefreshSecret)
	if err != nil {
		return nil, err
	}

	if err := h.Repo.AddRefreshToDB(refreshToken); err != nil {
		l.Error("internal error", "error", err)
		return nil, err
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
		IsAdmin:      user.Role == "admin",
	}, nil

}

func (h *AuthService) LogOut(ctx context.Context, refreshtoken string) error {
	result := h.Repo.DB.Model(&models.RefreshToken{}).
		Where("token = ?", jwthelp.Sha256Hex(refreshtoken)).
		Update("revoked", true)
	return result.Error
}
