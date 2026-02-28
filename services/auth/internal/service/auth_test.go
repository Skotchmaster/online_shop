package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Skotchmaster/online_shop/pkg/tokens"
	"github.com/Skotchmaster/online_shop/services/auth/internal/repo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuthService() *AuthService {
	return &AuthService{
		Repo: repo.GormRepo{
			JWTSecret:     []byte("test-jwt-secret"),
			RefreshSecret: []byte("test-refresh-secret"),
		},
	}
}

func TestAuthService_CreateAccessToken_SetsExpectedClaims(t *testing.T) {
	t.Parallel()

	svc := newTestAuthService()
	userID := uuid.NewString()
	role := "admin"
	accessExp := time.Now().Add(15 * time.Minute).UTC()

	token, err := svc.CreateAccessToken(role, userID, accessExp)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := tokens.AccessClaimsFromToken(token, svc.Repo.JWTSecret)
	require.NoError(t, err)

	assert.Equal(t, role, claims.Role)
	assert.Equal(t, userID, claims.Subject)
	require.NotNil(t, claims.ExpiresAt)
	assert.WithinDuration(t, accessExp, claims.ExpiresAt.Time, time.Second)
}

func TestAuthService_CreateRefreshToken_SetsExpectedClaims(t *testing.T) {
	t.Parallel()

	svc := newTestAuthService()
	userID := uuid.NewString()
	refreshExp := time.Now().Add(24 * time.Hour).UTC()

	token, err := svc.CreateRefreshToken(userID, refreshExp)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := tokens.RefreshClaimsFromToken(token, svc.Repo.RefreshSecret)
	require.NoError(t, err)

	assert.Equal(t, userID, claims.Subject)
	assert.NotEmpty(t, claims.ID)
	require.NotNil(t, claims.ExpiresAt)
	assert.WithinDuration(t, refreshExp, claims.ExpiresAt.Time, time.Second)
}

func TestAuthService_Register_Validation(t *testing.T) {
	t.Parallel()

	svc := newTestAuthService()
	ctx := context.Background()

	tests := []struct {
		name     string
		username string
		password string
	}{
		{name: "empty username", username: "", password: "secret"},
		{name: "empty password", username: "user", password: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := svc.Register(ctx, tt.username, tt.password)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestAuthService_Login_Validation(t *testing.T) {
	t.Parallel()

	svc := newTestAuthService()
	ctx := context.Background()

	tests := []struct {
		name     string
		username string
		password string
	}{
		{name: "empty username", username: "", password: "secret"},
		{name: "empty password", username: "user", password: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := svc.Login(ctx, tt.username, tt.password)
			require.Error(t, err)
			assert.Nil(t, res)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestAuthService_LogOut_EmptyToken_NoError(t *testing.T) {
	t.Parallel()

	svc := newTestAuthService()
	err := svc.LogOut(context.Background(), "")
	require.NoError(t, err)
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	t.Parallel()

	svc := newTestAuthService()
	res, err := svc.Refresh(context.Background(), "not-a-valid-jwt")

	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, ErrInvalidRefreshToken))
}
