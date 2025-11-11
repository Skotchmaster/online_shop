package httpserver

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/logging"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/services/auth/internal/service"
	"github.com/labstack/echo/v4"
)

type AuthHTTP struct {
	Svc *service.AuthService
}

func (h *AuthHTTP) Register(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "auth_register")
	var req struct {
		Username string
		Password string
	}

	if err := c.Bind(&req); err != nil {
		l.Warn("register_error", "status", 400, "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	if err := h.Svc.Register(ctx, req.Username, req.Password); err != nil {
		l.Warn("register_failed", "error", err)
		if err.Error() == "user already exist" {
			return echo.NewHTTPError(http.StatusConflict, "user already exists")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "registration failed")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"username": req.Username,
	})
}


func (h *AuthHTTP) Login(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "auth_login")

	var req struct {
		Username string
		Password string
	}

	if err := c.Bind(&req); err != nil {
		l.Warn("login_error", "status", 400, "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	res, err := h.Svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		l.Warn("login_failed", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}

	accessCookie := jwthelp.CreateCookie("accessToken", res.AccessToken, "/", res.AccessExp)
	c.SetCookie(accessCookie)

	refreshCookie := jwthelp.CreateCookie("refreshToken", res.RefreshToken, "/", res.RefreshExp)
	c.SetCookie(refreshCookie)
	l.Info("login_successful")

	return c.JSON(http.StatusOK, echo.Map{
		"is_admin": res.IsAdmin,
	})

}

func (h *AuthHTTP) LogOut(c echo.Context) error {
    ctx := c.Request().Context()
    l := logging.FromContext(ctx).With("handler", "auth_logout")

    var refreshTokenValue string
    
    refreshCookie, err := c.Cookie("refreshToken")
    if err == nil && refreshCookie != nil {
        refreshTokenValue = refreshCookie.Value
    }

    if err := h.Svc.LogOut(ctx, refreshTokenValue); err != nil {
        c.SetCookie(jwthelp.DeleteCookie("refreshToken", "/"))
        c.SetCookie(jwthelp.DeleteCookie("accessToken", "/"))
        
        l.Error("logout_failed", "status", 500, "error", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "logout failed")
    }

    c.SetCookie(jwthelp.DeleteCookie("refreshToken", "/"))
    c.SetCookie(jwthelp.DeleteCookie("accessToken", "/"))

    l.Info("logout_successful")
    return c.JSON(http.StatusOK, echo.Map{
        "message": "logged out",
    })
}

func (h *AuthHTTP) Refresh(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "refresh")

	refreshToken, err := c.Cookie("refreshToken")
	if err != nil {
		l.Warn("refresh_failed", "status", 401, "reason", "missing refresh token", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "missing refresh token")
	}

	accessToken, err := c.Cookie("accessToken")
	if err != nil {
		l.Warn("refresh_failed", "status", 401, "reason", "missing access token", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "missing access token")
	}

	res, err := h.Svc.Refresh(ctx, refreshToken.Value, accessToken.Value)
	if err != nil {
		c.SetCookie(jwthelp.DeleteCookie("refreshToken", "/"))
		c.SetCookie(jwthelp.DeleteCookie("accessToken", "/"))
		l.Warn("refresh_failed", "status", 401, "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "refresh failed")
	}
	accessCookie := jwthelp.CreateCookie("accessToken", res.AccessToken, "/", res.AccessExp)
	c.SetCookie(accessCookie)

	refreshCookie := jwthelp.CreateCookie("refreshToken", res.RefreshToken, "/", res.RefreshExp)
	c.SetCookie(refreshCookie)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  res.AccessToken,
		"refresh_token": res.RefreshToken,
		"access_exp":    res.AccessExp.Unix(),
		"refresh_exp":   res.RefreshExp.Unix(),
		"is_admin":      res.IsAdmin,
	})
}
 