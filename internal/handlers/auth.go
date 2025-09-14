package handlers

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

func CreateCookie(name string, value string, path string, exp_time time.Time) *http.Cookie {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Expires:  exp_time,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return cookie
}

func (h *AuthHandler) Register(c echo.Context) error {
	ctx := c.Request().Context()
	var req struct {
		Username string
		Password string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}

	hash, err := hash.HashPassword(req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         "user"}
	l := logging.FromContext(ctx).With("handler", "auth.register")
	var user_chek models.User
	result := h.DB.Where("username=?", req.Username).First(&user_chek)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
	} else {
		l.Warn("register_failed",
			"status", 409,
			"reason", "user_exists",
		)
		return echo.NewHTTPError(http.StatusConflict, "user already exists")
	}
	if err := h.DB.Create(&user).Error; err != nil {
		l.Warn("register_failed",
			"status", 401,
			"reason", "db_error",
		)
		return echo.NewHTTPError(http.StatusUnauthorized, err)
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
	user.PasswordHash = req.Password

	l.Info("register_success",
		"status", 200,
	)
	return c.JSON(http.StatusOK, user)

}

func (h *AuthHandler) Login(c echo.Context) error {
	var req struct {
		Username string
		Password string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "auth.register")
	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		l.Warn("login_failed",
			"status", 401,
			"reason", "invalid username or password",
		)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}

	if !hash.CheckPassword(user.PasswordHash, req.Password) {
		l.Warn("login_failed",
			"status", 401,
			"reason", "invalid username or password",
		)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}

	role := "user"
	if user.Role == "admin" {
		role = "admin"
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
		l.Warn("login_failed",
			"status", 401,
			"reason", err,
		)
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}

	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := jwt.MapClaims{
		"sub":  user.ID,
		"role": role,
		"exp":  refreshExp,
		"typ":  "refresh",
	}
	tokenRef := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := tokenRef.SignedString(h.RefreshSecret)
	if err != nil {
		l.Warn("login_failed",
			"status", 500,
			"reason", err,
		)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	refreshModel := models.RefreshToken{
		Role:      role,
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Time(refreshExp).Unix(),
		Revoked:   false,
	}

	if err := h.DB.Create(&refreshModel).Error; err != nil {
		l.Warn("login_failed",
			"status", 401,
			"reason", err,
		)
		return echo.NewHTTPError(http.StatusUnauthorized, err)
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
		"access_token":  accesToken,
		"refresh_token": refreshToken,
		"is_admin":      role == "admin",
	})

}

func (h *AuthHandler) LogOut(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "Auth.LogOut")

	refreshCookie, err := c.Cookie("refreshToken")
	if err != nil {
		l.Warn("logout_failed", "status", 400, "reason", "missing_refresh_cookie", "error", err)
		return c.JSON(http.StatusBadRequest, err)
	}

	result := h.DB.Model(&models.RefreshToken{}).
		Where("token = ?", refreshCookie.Value).
		Update("revoked", true)

	if result.Error != nil {
		l.Error("logout_failed", "status", 500, "reason", "db_error", "error", result.Error)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": result.Error.Error(),
		})
	}

	expired := time.Now().Add(-1 * time.Hour)

	c.SetCookie(CreateCookie("accessToken", "/", "/", expired))
	c.SetCookie(CreateCookie("refreshToken", "/", "/", expired))
	l.Info("successful_logout")
	return c.JSON(http.StatusOK, echo.Map{
		"message": "loged out",
	})
}
