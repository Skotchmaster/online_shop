package tests

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Skotchmaster/online_shop/pkg/tokens"
	"github.com/Skotchmaster/online_shop/services/auth/internal/models"
	"github.com/Skotchmaster/online_shop/services/auth/internal/repo"
	"github.com/Skotchmaster/online_shop/services/auth/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type integrationEnv struct {
	db  *gorm.DB
	svc *service.AuthService
	rp  repo.GormRepo
}

func newIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()

	dsn := os.Getenv("AUTH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AUTH_TEST_DATABASE_URL is required for tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.RefreshToken{}))

	rp := repo.GormRepo{
		DB:            db,
		JWTSecret:     []byte("test-jwt-secret"),
		RefreshSecret: []byte("test-refresh-secret"),
	}

	env := &integrationEnv{
		db: db,
		rp: rp,
		svc: &service.AuthService{
			Repo: rp,
		},
	}

	t.Cleanup(func() {
		truncateTables(t, db)
	})

	return env
}

func truncateTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	db.Exec("TRUNCATE TABLE refresh_tokens, users RESTART IDENTITY CASCADE")
}

func uniqueUsername() string {
	return "u_" + uuid.NewString()
}

func TestAuthService_Register_SuccessAndConflict(t *testing.T) {
	env := newIntegrationEnv(t)
	ctx := context.Background()
	username := uniqueUsername()

	err := env.svc.Register(ctx, username, "Secret123")
	require.NoError(t, err)

	err = env.svc.Register(ctx, username, "Secret123")
	require.Error(t, err)
	assert.ErrorIs(t, err, service.ErrConflict)
}

func TestAuthService_Login_Success_IssuesTokens(t *testing.T) {
	env := newIntegrationEnv(t)
	ctx := context.Background()
	username := uniqueUsername()

	require.NoError(t, env.svc.Register(ctx, username, "Secret123"))

	res, err := env.svc.Login(ctx, username, "Secret123")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res.AccessToken)
	require.NotEmpty(t, res.RefreshToken)
	assert.False(t, res.IsAdmin)

	accessClaims, err := tokens.AccessClaimsFromToken(res.AccessToken, env.rp.JWTSecret)
	require.NoError(t, err)
	assert.Equal(t, "user", accessClaims.Role)
	require.NotNil(t, accessClaims.ExpiresAt)
	assert.True(t, accessClaims.ExpiresAt.Time.After(time.Now().UTC()))

	refreshClaims, err := tokens.RefreshClaimsFromToken(res.RefreshToken, env.rp.RefreshSecret)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshClaims.ID)
	require.NotNil(t, refreshClaims.ExpiresAt)
	assert.True(t, refreshClaims.ExpiresAt.Time.After(time.Now().UTC()))
}

func TestAuthService_Refresh_Success_RotatesToken(t *testing.T) {
	env := newIntegrationEnv(t)
	ctx := context.Background()
	username := uniqueUsername()

	require.NoError(t, env.svc.Register(ctx, username, "Secret123"))
	loginRes, err := env.svc.Login(ctx, username, "Secret123")
	require.NoError(t, err)

	oldClaims, err := tokens.RefreshClaimsFromToken(loginRes.RefreshToken, env.rp.RefreshSecret)
	require.NoError(t, err)

	refreshed, err := env.svc.Refresh(ctx, loginRes.RefreshToken)
	require.NoError(t, err)
	require.NotNil(t, refreshed)
	require.NotEmpty(t, refreshed.AccessToken)
	require.NotEmpty(t, refreshed.RefreshToken)
	assert.NotEqual(t, loginRes.RefreshToken, refreshed.RefreshToken)

	oldTokenModel, err := env.rp.FindRefreshByID(ctx, oldClaims.ID)
	require.NoError(t, err)
	assert.True(t, oldTokenModel.Revoked)

	newClaims, err := tokens.RefreshClaimsFromToken(refreshed.RefreshToken, env.rp.RefreshSecret)
	require.NoError(t, err)
	newTokenModel, err := env.rp.FindRefreshByID(ctx, newClaims.ID)
	require.NoError(t, err)
	assert.False(t, newTokenModel.Revoked)
}

func TestAuthService_LogOut_RevokesRefreshToken(t *testing.T) {
	env := newIntegrationEnv(t)
	ctx := context.Background()
	username := uniqueUsername()

	require.NoError(t, env.svc.Register(ctx, username, "Secret123"))
	loginRes, err := env.svc.Login(ctx, username, "Secret123")
	require.NoError(t, err)

	claims, err := tokens.RefreshClaimsFromToken(loginRes.RefreshToken, env.rp.RefreshSecret)
	require.NoError(t, err)

	require.NoError(t, env.svc.LogOut(ctx, loginRes.RefreshToken))

	tokenModel, err := env.rp.FindRefreshByID(ctx, claims.ID)
	require.NoError(t, err)
	assert.True(t, tokenModel.Revoked)
}

func TestAuthService_Refresh_RevokedToken_ReturnsInvalidRefresh(t *testing.T) {
	env := newIntegrationEnv(t)
	ctx := context.Background()
	username := uniqueUsername()

	require.NoError(t, env.svc.Register(ctx, username, "Secret123"))
	loginRes, err := env.svc.Login(ctx, username, "Secret123")
	require.NoError(t, err)

	require.NoError(t, env.svc.LogOut(ctx, loginRes.RefreshToken))

	res, err := env.svc.Refresh(ctx, loginRes.RefreshToken)
	require.Error(t, err)
	assert.Nil(t, res)
	assert.True(t, errors.Is(err, service.ErrInvalidRefreshToken))
}
