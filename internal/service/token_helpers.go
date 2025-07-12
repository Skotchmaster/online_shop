package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func ValidateRefresh(rawToken string, Refreshsecret []byte, db *gorm.DB) (jwt.MapClaims, error) {
	t, err := jwt.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signature method: %v", t.Header["alg"])
		}
		return Refreshsecret, nil
	})

	if err != nil || !t.Valid {
		return nil, fmt.Errorf("invalid refresh token %w", err)
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("cannot parse claims")
	}

	if typ, ok := claims["typ"]; !ok || typ != "refresh" {
		return nil, fmt.Errorf("not a refresh token")
	}

	var stored models.RefreshToken
	if err := db.Where("token=?", rawToken).First(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token dont found")
		}
		return nil, fmt.Errorf("db error: %w", err)
	}

	if stored.Revoked {
		return nil, fmt.Errorf("refresh token revoked")
	}
	if time.Now().Unix() > stored.ExpiresAt {
		return nil, fmt.Errorf("refresh token expierd")
	}

	return claims, nil
}

func SingAccessToken(userID uint, role string, accessSecret []byte) (string, error) {
	exp := time.Now().Add(15 * time.Minute)
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  exp.Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(accessSecret)
}

func SingRefreshToken(userID uint, role string, refreshSecret []byte) (string, error) {
	exp := time.Now().Add(7 * 24 * time.Hour)
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  exp.Unix(),
		"typ":  "refresh",
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(refreshSecret)
}

func SaveRefreshToken(db *gorm.DB, token string, userID uint) error {
	exp := time.Now().Add(7 * 24 * time.Hour)
	new := models.RefreshToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: exp.Unix(),
		Revoked:   false,
	}
	if err := db.Create(&new).Error; err != nil {
		return fmt.Errorf("db error: %w", err)
	}

	return nil
}
