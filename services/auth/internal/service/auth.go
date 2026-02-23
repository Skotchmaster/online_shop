package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	pkg_hash "github.com/Skotchmaster/online_shop/pkg/hash"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/pkg/tokens"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"github.com/Skotchmaster/online_shop/services/auth/internal/repo"
	"github.com/Skotchmaster/online_shop/services/auth/internal/transport"
)

var (
	ErrValidation          = errors.New("validation")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrConflict            = errors.New("conflict")
	ErrInternal            = errors.New("internal error")
)

type AuthService struct {
	Repo repo.GormRepo
}

func (h *AuthService) CreateAccessToken(role, id string, accessExp time.Time) (string, error) {
	accessClaims := tokens.AccessClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   id,
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}

	tokenAccess := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := tokenAccess.SignedString(h.Repo.JWTSecret)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

func (h *AuthService) CreateRefreshToken(id string, refreshExp time.Time) (string, error) {
	jti := jwthelp.NewJTI()
	refreshClaims := tokens.RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   id,
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			ID:        jti,
		},
	}

	tokenRefresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := tokenRefresh.SignedString(h.Repo.RefreshSecret)
	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

func (h *AuthService) Register(ctx context.Context, username, password string) error {
	if username == "" {
		return fmt.Errorf("username must not be empty: %w", ErrValidation)
	}
	if password == "" {
		return fmt.Errorf("password must not be empty: %w", ErrValidation)
	}

	pwHash, err := pkg_hash.HashPassword(password)
	if err != nil {
		return fmt.Errorf("cannot hash the password: %w", ErrInternal)
	}
	user := models.User{
		Username:     username,
		PasswordHash: string(pwHash),
		Role:         "user",
	}

	if err := h.Repo.CreateUserIfNotExists(ctx, &user); err != nil {
		if errors.Is(err, repo.ErrUserAlreadyExist) {
			return fmt.Errorf("user already exist: %w", ErrConflict)
		} else {
			return fmt.Errorf("internal server error: %w", ErrInternal)
		}
	}
	return nil
}

func (h *AuthService) Login(ctx context.Context, username, password string) (*transport.LoginResult, error) {
	if username == "" {
		return nil, fmt.Errorf("username must not be empty: %w", ErrValidation)
	}
	if password == "" {
		return nil, fmt.Errorf("password must not be empty: %w", ErrValidation)
	}

	user, err := h.Repo.UserExist(ctx, username, password)
	if err != nil {
		if errors.Is(err, repo.ErrInvalidCredentials) {
			return nil, fmt.Errorf("invalid username or password: %w", ErrUnauthorized)
		}
		return nil, fmt.Errorf("internal server error: %w", ErrInternal)
	}

	accessExp := time.Now().Add(time.Minute * 15)
	accessToken, err := h.CreateAccessToken(user.Role, user.ID.String(), accessExp)
	if err != nil {
		return nil, fmt.Errorf("internal server error: %w", ErrInternal)
	}

	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshToken, err := h.CreateRefreshToken(user.ID.String(), refreshExp)
	if err != nil {
		return nil, fmt.Errorf("internal server error: %w", ErrInternal)
	}

	if err := h.Repo.AddRefreshToDB(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("internal server error: %w", ErrInternal)
	}

	return &transport.LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
		IsAdmin:      user.Role == "admin",
	}, nil

}

func (h *AuthService) LogOut(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	if err := h.Repo.LogOut(ctx, refreshToken); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}

func (h *AuthService) Refresh(ctx context.Context, refreshToken string) (*transport.LoginResult, error) {
	refresh_claims, err := tokens.RefreshClaimsFromToken(refreshToken, h.Repo.RefreshSecret)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	jti := refresh_claims.ID
	userId := refresh_claims.Subject
	userUuid, err := uuid.Parse(userId)

	if err != nil {
		return nil, fmt.Errorf("user id is not uuid: %w", ErrInvalidRefreshToken)
	}

	user, err := h.Repo.GetUserById(ctx, userUuid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user with this id dont exist: %w", ErrInvalidRefreshToken)
		} else {
			return nil, ErrInternal
		}
	}
	role := user.Role

	accessExp := time.Now().Add(time.Minute * 15)
	accessTokenNew, err := h.CreateAccessToken(role, userId, accessExp)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", ErrInternal)
	}

	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshTokenNew, err := h.CreateRefreshToken(userId, refreshExp)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", ErrInternal)
	}

	newRefreshClaims, err := tokens.RefreshClaimsFromToken(refreshTokenNew, h.Repo.RefreshSecret)
	if err != nil {
		return nil, ErrInternal
	}

	newRefreshModel := models.RefreshToken{
		Token:     jwthelp.Sha256Hex(refreshTokenNew),
		UserID:    userUuid,
		ExpiresAt: newRefreshClaims.ExpiresAt.Time,
		JTI:       newRefreshClaims.ID,
	}

	if err := h.Repo.RotateRefreshToken(ctx, jti, newRefreshModel); err != nil {
		if errors.Is(err, repo.ErrTokenExpiredOrRevoked) {
			return nil, fmt.Errorf("failed to rotate refresh token with jti: %s with error: %w", jti, ErrInvalidRefreshToken)
		}
		return nil, fmt.Errorf("failed to rotate refresh token with jti: %s with error: %w", jti, err)
	}

	return &transport.LoginResult{
		AccessToken:  accessTokenNew,
		RefreshToken: refreshTokenNew,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
		IsAdmin:      role == "admin",
	}, nil
}
