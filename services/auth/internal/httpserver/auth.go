package httpserver

import (
	"errors"
	"net/http"

	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/pkg/logging"
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
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.Bind(&req); err != nil {
		l.Warn("register_failed", "status", 400, "reason", "invalid body", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	if err := h.Svc.Register(ctx, req.Username, req.Password); err != nil {
		if errors.Is(err, service.ErrValidation) {
			l.Warn("register_failed", "status", 400, "reason", "invalid credentials", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "invalid credentials")
		} else {
			if errors.Is(err, service.ErrConflict) {
				l.Warn("register_failed", "status", 409, "reason", "user already exist", "error", err)
				return echo.NewHTTPError(http.StatusConflict, "user already exist")
			}
		}
		l.Error("register_failed", "status", 500, "reason", "registration failed", "error", err)
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
		l.Warn("login_failed", "status", 400, "reason", "invalid body", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	res, err := h.Svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized){
			l.Warn("login_failed", "status", 401, "reason", "invalid username or password", "error", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
		}else {
			if errors.Is(err, service.ErrValidation){
				l.Warn("login_failed", "status", 400, "reason", "validation error", "error", err)
				return echo.NewHTTPError(http.StatusBadRequest, "validation error")
			}
		}
		l.Error("login_failed", "status", 500, "reason", "internal error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
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
        
        l.Error("logout_failed", "status", 500, "reason", "internal error", "error", err)
        return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
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

	res, err := h.Svc.Refresh(ctx, refreshToken.Value)
	if err != nil {
		c.SetCookie(jwthelp.DeleteCookie("refreshToken", "/"))
		c.SetCookie(jwthelp.DeleteCookie("accessToken", "/"))
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			l.Warn("refresh_failed", "status", 401, "reason", "refresh failed", "error", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "refresh failed")
		} 
		l.Error("refresh_failed", "status", 500, "reason", "internal error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	accessCookie := jwthelp.CreateCookie("accessToken", res.AccessToken, "/", res.AccessExp)
	c.SetCookie(accessCookie)

	refreshCookie := jwthelp.CreateCookie("refreshToken", res.RefreshToken, "/", res.RefreshExp)
	c.SetCookie(refreshCookie)

	l.Info("refresh_successful")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  res.AccessToken,
		"refresh_token": res.RefreshToken,
		"access_exp":    res.AccessExp.Unix(),
		"refresh_exp":   res.RefreshExp.Unix(),
		"is_admin":      res.IsAdmin,
	})
}
 