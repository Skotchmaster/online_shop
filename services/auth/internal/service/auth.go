package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	pkg_hash "github.com/Skotchmaster/online_shop/pkg/hash"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/pkg/logging"
	"github.com/Skotchmaster/online_shop/pkg/tokens"
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
		if errors.Is(err, repo.ErrUserAlreadyExist) {
			l.Error("register_error", "status", 409, "reason", "user already exist")
			return errors.New("user already exist")
		} else {
			l.Error("register_error", "status", 500, "reason", "internal Server Error", "error", err)
			return errors.New("internal server error")
		}
	}
	return nil
}

func (h *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	l := logging.FromContext(ctx).With("svc", "auth.login", "username", username)
	user, err := h.Repo.UserExist(username, password)
	if err != nil {
		if errors.Is(err, repo.ErrInvalidCredentials) {
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
		return nil, errors.New("internal Server error")
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
		IsAdmin:      user.Role == "admin",
	}, nil

}

func (h *AuthService) LogOut(ctx context.Context, refreshToken string) error {
    l := logging.FromContext(ctx).With("svc", "auth.logout")
    
    if refreshToken == "" {
        l.Warn("logout_skipped", "reason", "no refresh token provided")
        return nil
    }

    if err := h.Repo.LogOut(refreshToken); err != nil {
        l.Error("logout_failed", "status", 500, "reason", "failed to revoke refresh token", "error", err)
        return errors.New("internal server error")
    }
    
    l.Info("logout_successful")
    return nil
}

func(h *AuthService) Refresh(ctx context.Context, refreshToken, accessToken string) (*LoginResult, error) {
    l := logging.FromContext(ctx).With("svc", "auth.refresh")
    
    refresh_claims, err := tokens.RefreshClaimsFromToken(refreshToken, h.Repo.RefreshSecret)
    if err != nil {
        l.Warn("refresh_failed", "status", 401, "reason", "invalid refresh token", "error", err)
        return nil, errors.New("invalid refresh token")
    }
    
    access_claims, err := tokens.AccessClaimsFromToken(accessToken, h.Repo.JWTSecret)
    if err != nil {
        l.Warn("refresh_failed", "status", 401, "reason", "invalid access token", "error", err)
        return nil, errors.New("invalid access token")
    }
    
    jti := refresh_claims.ID
    userId := refresh_claims.Subject
    role := access_claims.Role
    
    accessExp := time.Now().Add(time.Minute * 15)
    accessTokenNew, err := h.CreateAccessToken(role, userId, accessExp)
    if err != nil {
        l.Error("refresh_failed", "status", 500, "reason", "failed to create access token", "error", err)
        return nil, errors.New("internal server error")
    }

    refreshExp := time.Now().Add(7 * 24 * time.Hour)
    refreshTokenNew, err := h.CreateRefreshToken(userId, refreshExp)
    if err != nil {
        l.Error("refresh_failed", "status", 500, "reason", "failed to create refresh token", "error", err)
        return nil, errors.New("internal server error")
    }

    newRefreshClaims, err := tokens.RefreshClaimsFromToken(refreshTokenNew, h.Repo.RefreshSecret)
    if err != nil {
        l.Error("refresh_failed", "status", 500, "reason", "failed to parse new refresh token", "error", err)
        return nil, errors.New("internal server error")
    }

    userID, err := uuid.Parse(userId)
    if err != nil {
        l.Error("refresh_failed", "status", 500, "reason", "invalid user id format", "user_id", userId, "error", err)
        return nil, errors.New("internal server error")
    }

    newRefreshModel := models.RefreshToken{
        Token:     jwthelp.Sha256Hex(refreshTokenNew),
        UserID:    userID,
        ExpiresAt: newRefreshClaims.ExpiresAt.Time.Unix(),
        JTI:       newRefreshClaims.ID,
    }

    if err := h.Repo.RotateRefreshToken(jti, newRefreshModel); err != nil {
        l.Error("refresh_failed", "status", 500, "reason", "failed to rotate refresh token", "jti", jti, "error", err)
        return nil, errors.New("internal server error")
    }

    l.Info("refresh_successful", "user_id", userId)
    return &LoginResult{
        AccessToken:  accessTokenNew,
        RefreshToken: refreshTokenNew,
        AccessExp:    accessExp,
        RefreshExp:   refreshExp,
        IsAdmin:      role == "admin",
    }, nil
}