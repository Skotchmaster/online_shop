package handlers

import (
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

func GetID(c echo.Context) (uint, error) {
	tok, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return 0, c.JSON(http.StatusBadRequest, "invalid token")
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return 0, c.JSON(http.StatusBadRequest, "invalid token")
	}

	id, ok := claims["sub"].(float64)
	if !ok {
		return 0, c.JSON(http.StatusBadRequest, "invalid token")
	}

	return uint(id), nil
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req struct {
		Username string
		Password string
		role     string
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error)
	}

	hash, _ := hash.HashPassword(req.Password)
	user := models.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         req.role}
	if err := h.DB.Create(&user).Error; err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":     "user_registrated",
		"UserID":   user.ID,
		"username": user.Username,
	}
	if err := h.Producer.PublishEvent(
		c.Request().Context(),
		"product_events",
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
		return c.JSON(http.StatusBadRequest, err)
	}

	var user models.User
	if err := h.DB.Where("username?", req.Username).First(&user).Error; err != nil {
		return c.JSON(http.StatusBadRequest, "invalid username or password")
	}

	if hash.ChekPassword(user.PasswordHash, req.Password) != true {
		return c.JSON(http.StatusUnauthorized, "invalid username or password")
	}

	role := "user"
	if user.Role == "admin" {
		role = "admin"
	}

	accesExp := time.Now().Add(time.Minute * 15).Unix()
	accesClaims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  accesExp,
	}

	tokenAcces := jwt.NewWithClaims(jwt.SigningMethodHS256, accesClaims)
	accesToken, err := tokenAcces.SignedString(h.JWTSecret)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	refreshExp := time.Now().Add(7 * 24 * time.Hour).Unix()
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
		ExpiresAt: time.Unix(accesExp, 0),
		Revoked:   false,
	}

	if err := h.DB.Create(&refreshModel); err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	event := map[string]interface{}{
		"type":     "user_loged_in",
		"UserID":   user.ID,
		"username": user.Username,
	}
	if err := h.Producer.PublishEvent(
		c.Request().Context(),
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
