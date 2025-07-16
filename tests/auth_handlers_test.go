package tests

import (
	"encoding/json"
	"net/http"

	"testing"

	"github.com/Skotchmaster/online_shop/internal/hash"
	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	payload := map[string]string{
		"username": "test_user",
		"password": "password",
	}
	env := newTestEnv(t)
	rec, _, c := env.doJSONRequest(http.MethodPost, "/register", payload)

	require.NoError(t, env.A.Register(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var valid_user models.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &valid_user))
	require.Equal(t, "test_user", valid_user.Username)
	require.Equal(t, "user", valid_user.Role)
	require.NotEmpty(t, valid_user.ID)
	require.NotEqual(t, "password", valid_user.PasswordHash)

	_, _, c_invalid := env.doJSONRequest(http.MethodPost, "/register", payload)

	err := env.A.Register(c_invalid)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok, "expected HTTPError")
	require.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestLogin(t *testing.T) {
	env := newTestEnv(t)

	hash, _ := hash.HashPassword("password")
	user := models.User{
		Username:     "test_user",
		PasswordHash: hash,
		Role:         "user",
	}
	env.DB.Create(&user)

	load := map[string]string{
		"username": "test_user",
		"password": "password",
	}
	rec, _, c := env.doJSONRequest(http.MethodPost, "/login", load)

	require.NoError(t, env.A.Login(c))
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

	_, _, c_invalid := env.doJSONRequest(http.MethodPost, "/login", invalid_load)

	err := env.A.Login(c_invalid)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok, "expected HTTPError")
	require.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestLogOut(t *testing.T) {
	env := newTestEnv(t)

	load := map[string]string{
		"username": "test_user",
		"password": "password",
	}

	rec, _, c := env.doJSONRequest(http.MethodPost, "/register", load)
	require.NoError(t, env.A.Register(c))
	require.Equal(t, http.StatusOK, rec.Code)

	rec_login, _, c_login := env.doJSONRequest(http.MethodPost, "/login", load)
	require.NoError(t, env.A.Login(c_login))
	require.Equal(t, http.StatusOK, rec_login.Code)

	var RespData_login map[string]interface{}
	require.NoError(t, json.Unmarshal(rec_login.Body.Bytes(), &RespData_login))
	refresh_token := RespData_login["refresh_token"]

	ck := &http.Cookie{
		Name:  "refreshToken",
		Value: refresh_token.(string),
	}
	rec_logout, _, c_logout := env.doJSONRequest(http.MethodPost, "/logout", nil, ck)

	require.NoError(t, env.A.LogOut(c_logout))
	require.Equal(t, http.StatusOK, rec_logout.Code)

	var RespData_logout map[string]string
	require.NoError(t, json.Unmarshal(rec_logout.Body.Bytes(), &RespData_logout))
	require.Equal(t, "logged out", RespData_logout["message"])
}
