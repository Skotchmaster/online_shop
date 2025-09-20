package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"testing"

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
	require.Equal(t, http.StatusConflict, he.Code)
}

func TestLogin(t *testing.T) {
    env := newTestEnv(t)

    access, refresh := login(t, env)

    require.NotEmpty(t, access)
    require.NotEmpty(t, refresh)
}


func TestLogOut(t *testing.T) {
    env := newTestEnv(t)

    access, refresh := login(t, env)
    recTok, _, _ := env.doJSONRequest(http.MethodGet, "/api/v1/products", nil)
    require.Equal(t, http.StatusOK, recTok.Code)

    var csrf string
    for _, ck := range recTok.Result().Cookies() {
        if ck.Name == "csrf_token" {
            csrf = ck.Value
            break
        }
    }
    require.NotEmpty(t, csrf, "csrf_token must be set by CSRF middleware on GET")

    rec, _, c := env.doJSONRequest(
        http.MethodPost,
        "/logout",
        nil,
        &http.Cookie{Name: "accessToken",  Value: access,  Path: "/"},
        &http.Cookie{Name: "refreshToken", Value: refresh, Path: "/"},
        &http.Cookie{Name: "csrf_token",   Value: csrf,    Path: "/"},
    )
    c.Request().Header.Set("X-CSRF-Token", csrf)
    c.Request().Header.Set("Origin",  "http://localhost")
    c.Request().Header.Set("Referer", "http://localhost/")

    require.NoError(t, env.A.LogOut(c))
    require.Equal(t, http.StatusOK, rec.Code)

    var accessDeleted, refreshDeleted bool
    now := time.Now().Add(2 * time.Second)
    for _, ck := range rec.Result().Cookies() {
        switch ck.Name {
        case "accessToken":
            accessDeleted = ck.Value == "" && (ck.MaxAge <= 0 || ck.Expires.Before(now))
        case "refreshToken":
            refreshDeleted = ck.Value == "" && (ck.MaxAge <= 0 || ck.Expires.Before(now))
        }
    }
    require.True(t, accessDeleted,  "accessToken must be deleted")
    require.True(t, refreshDeleted, "refreshToken must be deleted")
}