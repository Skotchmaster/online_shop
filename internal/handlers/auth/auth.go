package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Skotchmaster/online_shop/internal/hash"
	"github.com/Skotchmaster/online_shop/internal/logging"
	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
)

type AuthHandler struct {
	DB            *gorm.DB
	JWTSecret     []byte
	RefreshSecret []byte
	Producer      *mykafka.Producer
}

func (h *AuthHandler) Register(c echo.Context) error {
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

	pwHash, err := hash.HashPassword(req.Password)
	if err != nil {
		l.Error("register_error", "status", 500, "reason", "cannot hash the password", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "cannot hash the password")
	}
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(pwHash),
		Role:         "user"}
	var userCheck models.User
	if err := h.DB.Where("username = ?", req.Username).First(&userCheck).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			l.Error("register_error", "status", 500, "reason", "db_error", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
	} else {
		l.Warn("register_failed", "status", 409, "reason", "user_exists")
		return echo.NewHTTPError(http.StatusConflict, "user already exists")
	}
	if err := h.DB.Create(&user).Error; err != nil {
		l.Error("register_failed", "status", 500, "reason", "db_error", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}

	event := map[string]interface{}{
		"type":     "user_registrated",
		"UserID":   user.ID,
		"username": user.Username,
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	if err := h.Producer.PublishEvent(
		ctx,
		"user_events",
		fmt.Sprint(user.ID),
		event,
	); err != nil {
		c.Logger().Errorf("Kafka publish error: %v", err)
	}

	l.Info("register_success", "status", 200)
	return c.JSON(http.StatusOK, echo.Map{
		"id": user.ID, "username": user.Username, "role": user.Role,
	})

}

func (h *AuthHandler) Login(c echo.Context) error {
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
	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		l.Warn("login_failed", "status", 401, "reason", "invalid username or password",)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}

	if !hash.CheckPassword(user.PasswordHash, req.Password) {
		l.Warn("login_failed", "status", 401, "reason", "invalid username or password")
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}

	accessExp := time.Now().Add(time.Minute * 15)
	accessClaims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  accessExp.Unix(),
	}

	tokenAcces := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accesToken, err := tokenAcces.SignedString(h.JWTSecret)
	if err != nil {
		l.Error("login_failed", "status", 500, "reason", "cannot create token", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot create token")
	}

	jti := newJTI()
	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  refreshExp.Unix(),
		"typ":  "refresh",
		"jti":  jti,
	}
	tokenRef := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := tokenRef.SignedString(h.RefreshSecret)
	if err != nil {
		l.Error("login_failed", "status", 500, "reason", "cannot create token", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	refreshModel := models.RefreshToken{
		Role:      user.Role,
		Token:     sha256Hex(refreshToken),
		UserID:    user.ID,
		JTI:       jti,
		ExpiresAt: refreshExp.Unix(),
		Revoked:   false,
	}

	if err := h.DB.Create(&refreshModel).Error; err != nil {
		l.Warn("login_failed", "status", 500, "reason", "cannot add token to db")
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	accessCookie := CreateCookie("accessToken", accesToken, "/", accessExp)
	c.SetCookie(accessCookie)

	refreshCookie := CreateCookie("refreshToken", refreshToken, "/", refreshExp)
	l.Info("login_successful")
	c.SetCookie(refreshCookie)
	event := map[string]interface{}{
		"type":     "user_loged_in",
		"UserID":   user.ID,
		"username": user.Username,
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	if err := h.Producer.PublishEvent(
		ctx,
		"user_events",
		fmt.Sprint(user.ID),
		event,
	); err != nil {
		c.Logger().Errorf("Kafka publish error: %v", err)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"is_admin":      user.Role == "admin",
	})

}

func (h *AuthHandler) LogOut(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "auth_logout")

	refreshCookie, err := c.Cookie("refreshToken")
	if err != nil {
		l.Warn("logout_failed", "status", 401, "reason", "missing_refresh_cookie", "error", err)
	}

	result := h.DB.Model(&models.RefreshToken{}).
		Where("token = ?", sha256Hex(refreshCookie.Value)).
		Update("revoked", true)

	if result.Error != nil {
		l.Error("logout_failed", "status", 500, "reason", "cannot revoke refreshToken", "error", result.Error)
	}

	c.SetCookie(DeleteCookie("refreshToken", "/"))
	c.SetCookie(DeleteCookie("accessToken", "/"))
	l.Info("successful_logout")
	return c.JSON(http.StatusOK, echo.Map{
		"message": "loged out",
	})
}
