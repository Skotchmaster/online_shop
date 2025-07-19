package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Skotchmaster/online_shop/internal/handlers"
	"github.com/Skotchmaster/online_shop/internal/handlers/cart"
	"github.com/Skotchmaster/online_shop/internal/hash"
	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testEnv struct {
	T                        *testing.T
	E                        *echo.Echo
	A                        *handlers.AuthHandler
	C                        *cart.CartHandler
	P                        *handlers.ProductHandler
	DB                       *gorm.DB
	JWTSecret, RefreshSecret []byte
}

func InitTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to in-memory db: %v", err)
		return nil
	}

	if err := db.AutoMigrate(&models.CartItem{}, &models.Product{}, &models.RefreshToken{}, &models.User{}, &models.Order{}, &models.OrderItem{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	return db
}

func LoadConfig(t *testing.T) (*gorm.DB, []byte, []byte) {
	db := InitTestDB(t)

	jwt_secret := []byte(os.Getenv("JWT_SECRET"))
	refresh := []byte(os.Getenv("REFRESH_SECRET"))

	return db, jwt_secret, refresh
}

func newTestEnv(t *testing.T) *testEnv {
	db, jwt, refresh := LoadConfig(t)
	a := &handlers.AuthHandler{
		DB:            db,
		JWTSecret:     jwt,
		RefreshSecret: refresh,
		Producer:      &mykafka.Producer{},
	}
	c := &cart.CartHandler{
		DB:        db,
		JWTSecret: jwt,
		Producer:  &mykafka.Producer{},
	}
	p := &handlers.ProductHandler{
		DB:        db,
		JWTSecret: jwt,
		Producer:  &mykafka.Producer{},
	}
	return &testEnv{T: t, E: echo.New(), A: a, C: c, P: p, DB: db, JWTSecret: jwt, RefreshSecret: refresh}
}

func (env *testEnv) doJSONRequest(method, path string, body interface{}, cookies ...*http.Cookie) (*httptest.ResponseRecorder, []byte, echo.Context) {
	var buf bytes.Buffer
	if body != nil {
		require.NoError(env.T, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	c := env.E.NewContext(req, rec)
	return rec, rec.Body.Bytes(), c
}

func login(t *testing.T, env *testEnv) (string, string) {

	loadmap := map[string]string{
		"username": "username",
		"password": "password",
	}
	rec_register, _, c_register := env.doJSONRequest(http.MethodPost, "/register", loadmap)
	require.NoError(t, env.A.Register(c_register))
	require.Equal(t, http.StatusOK, rec_register.Code)

	rec_login, _, c_login := env.doJSONRequest(http.MethodPost, "/login", loadmap)
	require.NoError(t, env.A.Login(c_login))
	require.Equal(t, http.StatusOK, rec_login.Code)

	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(rec_login.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)

	return resp.AccessToken, resp.RefreshToken
}

func login_admin(t *testing.T, env *testEnv) (string, string) {
	hash, _ := hash.HashPassword("test_password")
	loadmap := models.User{
		Username:     "test_user",
		PasswordHash: hash,
		Role:         "admin",
	}

	env.DB.Create(&loadmap)

	new_loadmad := map[string]string{
		"username": "test_user",
		"password": "test_password",
	}

	rec_login, _, c_login := env.doJSONRequest(http.MethodPost, "/login", new_loadmad)
	require.NoError(t, env.A.Login(c_login))
	require.Equal(t, http.StatusOK, rec_login.Code)

	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(rec_login.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)

	return resp.AccessToken, resp.RefreshToken
}
