package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Skotchmaster/online_shop/internal/hash"
	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func InitTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to in-memory db: %v", err)
		return nil
	}

	if err := db.AutoMigrate(&models.CartItem{}, &models.Product{}, &models.RefreshToken{}, &models.User{}); err != nil {
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

func TestRegister(t *testing.T) {
	payload := map[string]string{
		"username": "test_user",
		"password": "password",
	}
	db, jwt_secret, refresh := LoadConfig(t)
	bodyBytes, _ := json.Marshal(payload)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	AuthHandler := AuthHandler{
		DB:            db,
		JWTSecret:     jwt_secret,
		RefreshSecret: refresh,
		Producer:      &mykafka.Producer{},
	}

	require.NoError(t, AuthHandler.Register(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var valid_user models.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &valid_user))
	require.Equal(t, "test_user", valid_user.Username)
	require.Equal(t, "user", valid_user.Role)
	require.NotEmpty(t, valid_user.ID)
	require.NotEqual(t, "password", valid_user.PasswordHash)

	req_invalid := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyBytes))
	req_invalid.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec_invalid := httptest.NewRecorder()
	c_invalid := e.NewContext(req_invalid, rec_invalid)

	err := AuthHandler.Register(c_invalid)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok, "expected HTTPError")
	require.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestLogin(t *testing.T) {
	db, jwt_secret, refresh := LoadConfig(t)

	AuthHandler := AuthHandler{
		DB:            db,
		JWTSecret:     jwt_secret,
		RefreshSecret: refresh,
		Producer:      &mykafka.Producer{},
	}

	hash, _ := hash.HashPassword("password")

	user := models.User{
		Username:     "test_user",
		PasswordHash: hash,
		Role:         "user",
	}

	db.Create(&user)

	load := map[string]string{
		"username": "test_user",
		"password": "password",
	}
	e := echo.New()
	bodyBytes, _ := json.Marshal(load)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, AuthHandler.Login(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var RespData map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &RespData))
	access_token, ok1 := RespData["access_token"]
	refresh_token, ok2 := RespData["refresh_token"]
	require.True(t, ok1, "expected 'access_token' in field")
	require.True(t, ok2, "expected 'refresh_token' in field")
	require.NotEmpty(t, access_token)
	require.NotEmpty(t, refresh_token)

	invalid_load := map[string]string{
		"username": "test_user",
		"password": "invalid_password",
	}

	badBodyBytes, _ := json.Marshal(invalid_load)
	req_invalid := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(badBodyBytes))
	req_invalid.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec_invalid := httptest.NewRecorder()
	c_invalid := e.NewContext(req_invalid, rec_invalid)

	err := AuthHandler.Login(c_invalid)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok, "expected HTTPError")
	require.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestLogOut(t *testing.T) {
	db, jwt_secret, refresh_secret := LoadConfig(t)
	AuthHandler := AuthHandler{
		DB:            db,
		JWTSecret:     jwt_secret,
		RefreshSecret: refresh_secret,
		Producer:      &mykafka.Producer{},
	}

	load := map[string]string{
		"username": "test_user",
		"password": "password",
	}

	e := echo.New()
	BodyBytes, _ := json.Marshal(load)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(BodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	require.NoError(t, AuthHandler.Register(c))
	require.Equal(t, http.StatusOK, rec.Code)

	req_login := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(BodyBytes))
	req_login.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec_login := httptest.NewRecorder()
	c_login := e.NewContext(req_login, rec_login)
	require.NoError(t, AuthHandler.Login(c_login))
	require.Equal(t, http.StatusOK, rec_login.Code)
	var RespData_login map[string]interface{}
	require.NoError(t, json.Unmarshal(rec_login.Body.Bytes(), &RespData_login))
	refresh_token := RespData_login["refresh_token"]

	req_logout := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req_logout.AddCookie(&http.Cookie{
		Name:  "refreshToken",
		Value: refresh_token.(string),
	})
	rec_logout := httptest.NewRecorder()
	c_logout := e.NewContext(req_logout, rec_logout)

	require.NoError(t, AuthHandler.LogOut(c_logout))
	require.Equal(t, http.StatusOK, rec_logout.Code)

	var RespData map[string]string
	require.NoError(t, json.Unmarshal(rec_logout.Body.Bytes(), &RespData))
	require.Equal(t, "logged out", RespData["message"])
}
