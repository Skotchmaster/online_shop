package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	pkg_hash "github.com/Skotchmaster/online_shop/pkg/hash"
	"github.com/Skotchmaster/online_shop/pkg/logging"
	"github.com/Skotchmaster/online_shop/pkg/tokens"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
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

func(h *AuthService) CreateAccessToken(role, id string, accessExp time.Time) (string, error) {
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

func(h *AuthService) CreateRefreshToken(id string, refreshExp time.Time) (string, error) {
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
	l := logging.FromContext(ctx).With("svc", "auth.register")

	pwHash, err := pkg_hash.HashPassword(password)
	if err != nil {
		l.Error("register_error", "status", 500, "reason", "cannot hash the password", "error", err)
		return err
	}
	user := models.User{
		Username:     username,
		PasswordHash: string(pwHash),
		Role:         "user",
	}
	
	if err := h.Repo.UserNotExist(user); err != nil{
		if errors.Is(err, errors.New("user already exist")) {
			l.Error("register_error", "status", 409, "reason", "user already exist")
			return errors.New("user already exist")
		} else {
			l.Error("register_error", "status", 500, "reason", "internal Server Error", "error", err)
			return errors.New("user already exist")
		}
	}
	return nil
}

func (h *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	l := logging.FromContext(ctx).With("svc", "auth.login", "username", username)
	user, err := h.Repo.UserExist(username, password)
	if err != nil {
		if errors.Is(err, errors.New("invalid credentials")) {
			l.Warn("login failed", "status", 422, "reason", "invalid username or password")
			return nil, errors.New("invalid username or password")
		}
		l.Warn("login failed", "status", 500, "error", err)
		return nil, errors.New("internal Server error")
	}

	accessExp := time.Now().Add(time.Minute * 15)
	accessToken, err := h.CreateAccessToken(user.Role, user.ID.String(), accessExp)
	if err != nil {
		l.Warn("login failed", "status", 500, "error", err)
		return nil, errors.New("internal Server error")
	}

	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshToken, err := h.CreateRefreshToken(user.ID.String(), refreshExp)
	if err != nil {
		l.Warn("login failed", "status", 500, "error", err)
		return nil, errors.New("internal Server error")
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

func(h *AuthService) Refresh(ctx context.Context, refreshToken, accessToken string) (*LoginResult, error) {
	l := logging.FromContext(ctx).With("svc", "auth.refresh")
	refresh_claims, err := tokens.RefreshClaimsFromToken(refreshToken, h.Repo.RefreshSecret)
	if err != nil {
		return nil, err
	}
	access_claims, err := tokens.AccessClaimsFromToken(accessToken, h.Repo.JWTSecret)
	if err != nil {
		return nil, err
	}
	jti := refresh_claims.ID
	if check, err := h.Repo.RefreshExists(jti); err != nil {
		return nil, err
	} else {
		if !check {
			return nil, errors.New("refreshToken not found")
		}
	}

	if check, err := h.Repo.RefreshExpiredOrRevoked(jti); err != nil {
		return nil, err
	} else {
		if check {
			return nil, errors.New("token expired or revoked")
		}
	}

	userId, role := refresh_claims.ID, access_claims.Role
	accessExp := time.Now().Add(time.Minute * 15)
	accessTokenNew, err := h.CreateAccessToken(userId, role, accessExp)
	if err != nil {
		l.Warn("login failed", "status", 500, "error", err)
		return nil, errors.New("internal Server error")
	}

	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshTokenNew, err := h.CreateRefreshToken(userId, refreshExp)
	if err != nil {
		l.Warn("login failed", "status", 500, "error", err)
		return nil, errors.New("internal Server error")
	}

	return &LoginResult{
		AccessToken:  accessTokenNew,
		RefreshToken: refreshTokenNew,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
		IsAdmin:      role == "admin",
	}, nil
}
