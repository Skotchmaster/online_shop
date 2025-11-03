package httpserver

import (
	"errors"
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/logging"
	jwthelp "github.com/Skotchmaster/online_shop/services/auth/internal/jwt"
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
		return echo.NewHTTPError(http.StatusUnauthorized, "regiser failed")
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
		code := http.StatusUnauthorized
		if errors.Is(err, echo.ErrBadGateway.Internal) {
			code = http.StatusInternalServerError
		}
		l.Warn("login_failed", "status", code, "error", err)
		return echo.NewHTTPError(code, "invalid username or password")
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

	refreshCookie, err := c.Cookie("refreshToken")
	if err == nil {
		
		result := h.Svc.LogOut(ctx, refreshCookie.Value)

		if result != nil {
			c.SetCookie(jwthelp.DeleteCookie("refreshToken", "/"))
			c.SetCookie(jwthelp.DeleteCookie("accessToken", "/"))
			l.Error("logout_failed", "status", 500, "reason", "cannot revoke refreshToken", "error", result)
			return echo.NewHTTPError(http.StatusInternalServerError, 500)
		}
	}
	
	c.SetCookie(jwthelp.DeleteCookie("refreshToken", "/"))
	c.SetCookie(jwthelp.DeleteCookie("accessToken", "/"))

	l.Info("successful_logout")
	return c.JSON(http.StatusOK, echo.Map{
		"message": "loged out",
	})
}
