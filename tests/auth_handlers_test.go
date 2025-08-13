package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

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

	payload2 := map[string]string{
		"username": fmt.Sprintf("test_user_%d", time.Now().UnixNano()),
		"password": "password",
	}
	event := consumeNextEvent(t, "user_events", func() {
		rec2, _, c2 := env.doJSONRequest(http.MethodPost, "/register", payload2)
		require.NoError(t, env.A.Register(c2))
		require.Equal(t, http.StatusOK, rec2.Code)
	})
	require.Equal(t, "user_registrated", event["type"])
	require.Equal(t, payload2["username"], event["username"])

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

	load := map[string]string{"username": "test_user", "password": "password"}

	var rec *httptest.ResponseRecorder
	event := consumeNextEvent(t, "user_events", func() {
		var c echo.Context
		rec, _, c = env.doJSONRequest(http.MethodPost, "/login", load)
		require.NoError(t, env.A.Login(c))
		require.Equal(t, http.StatusOK, rec.Code)
	})

	var RespData map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &RespData))
	require.NotEmpty(t, RespData["access_token"])
	require.NotEmpty(t, RespData["refresh_token"])

	require.Equal(t, "user_loged_in", event["type"])
	require.Equal(t, "test_user", event["username"])
}

func TestLogOut(t *testing.T) {
	env := newTestEnv(t)

	load := map[string]string{"username": "test_user", "password": "password"}

	recReg, _, cReg := env.doJSONRequest(http.MethodPost, "/register", load)
	require.NoError(t, env.A.Register(cReg))
	require.Equal(t, http.StatusOK, recReg.Code)

	recLogin, _, cLogin := env.doJSONRequest(http.MethodPost, "/login", load)
	require.NoError(t, env.A.Login(cLogin))
	require.Equal(t, http.StatusOK, recLogin.Code)

	var respLogin map[string]interface{}
	require.NoError(t, json.Unmarshal(recLogin.Body.Bytes(), &respLogin))
	ck := &http.Cookie{Name: "refreshToken", Value: respLogin["refresh_token"].(string)}
	recLogout, _, cLogout := env.doJSONRequest(http.MethodPost, "/logout", nil, ck)
	require.NoError(t, env.A.LogOut(cLogout))
	require.Equal(t, http.StatusOK, recLogout.Code)

	var respLogout map[string]string
	require.NoError(t, json.Unmarshal(recLogout.Body.Bytes(), &respLogout))
	require.Equal(t, "loged out", respLogout["message"])
}
