package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Skotchmaster/online_shop/internal/hash"
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
	var req struct {
		Username string
		Password string
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	hash, _ := hash.HashPassword(req.Password)
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hash)}
	if err := h.DB.Create(&user).Error; err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
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

	return c.JSON(http.StatusOK, user)

}

func (h *AuthHandler) Login(c echo.Context) error {
	var req struct {
		Username string
		Password string
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return c.JSON(http.StatusBadRequest, "invalid username or password")
	}

	if !hash.ChekPassword(user.PasswordHash, req.Password) {
		return c.JSON(http.StatusUnauthorized, "invalid username or password")
	}

	role := "user"
	if user.Role == "admin" {
		role = "admin"
	}

	accessExp := time.Now().Add(time.Minute * 15)
	accessClaims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  accessExp,
	}

	tokenAcces := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	tokenAcces.Header["kid"] = "v1"
	accesToken, err := tokenAcces.SignedString(h.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	refreshExp := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := jwt.MapClaims{
		"sub": user.ID,
		"exp": refreshExp,
		"typ": "refresh",
	}
	tokenRef := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := tokenRef.SignedString(h.RefreshSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "could not create refresh token"})
	}

	refreshModel := models.RefreshToken{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Time(refreshExp),
		Revoked:   false,
	}

	if err := h.DB.Create(&refreshModel).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	accessCookie := CreateCookie("accesToken", accesToken, "/", accessExp)
	c.SetCookie(accessCookie)

	refreshCookie := CreateCookie("refreshToken", refreshToken, "/", refreshExp)
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
