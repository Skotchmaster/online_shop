package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type TokenService struct {
	DB            *gorm.DB
	RefreshSecret []byte
	JWTSecret     []byte
}

func (t *TokenService) CheckCookie(c echo.Context) (string, string, string, error) {
	asCookie, err := c.Cookie("accessToken")
	if err == nil {
		token, err := jwt.Parse(asCookie.Value, func(j *jwt.Token) (interface{}, error) {
			if j.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected alg: %s", j.Method.Alg())
			}
			return t.JWTSecret, nil
		})
		if err == nil && token != nil && token.Valid {
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return "", "", "", fmt.Errorf("cannot parse claims")
			}	
			role, ok := claims["role"].(string)
			if !ok {
				return "", "", "", fmt.Errorf("cannot parse role")
			}
			setUserContext(c, claims)
			return asCookie.Value, "", role, nil
		}
		if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
            return "", "", "", fmt.Errorf("invalid access token: %w", err)
        }
	}

	rfCookie, err := c.Cookie("refreshToken")
	if err != nil {
		return "", "", "", err
	}
	newAccess, newRefresh, claims, err := t.RotateToken(rfCookie.Value)
	if err != nil {
		return "", "", "", err
	}

	role, ok := claims["role"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("cannot parse role")
	}

	setUserContext(c, claims)
	return newAccess, newRefresh, role, nil

}

func (t *TokenService) RotateToken(rawToken string) (string, string, jwt.MapClaims, error) {
	claims, err := ValidateRefresh(rawToken, t.RefreshSecret, t.DB)
	if err != nil {
		return "", "", nil, err
	}

	sub, ok := claims["sub"].(float64)
	if !ok { return "", "", nil, fmt.Errorf("invalid sub claim") }
	userID := uint(sub)

	role, ok := claims["role"].(string)
	if !ok { return "", "", nil, fmt.Errorf("invalid role claim") }

	var newAccess, newRefresh string

	if txErr := t.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.RefreshToken{}).
			Where("token = ?", rawToken).
			Update("revoked", true).Error; err != nil {
			return fmt.Errorf("revoke old refresh: %w", err)
		}

		newAccess, err = SignAccessToken(userID, role, t.JWTSecret)
		if err != nil { return err }

		newRefresh, err = SignRefreshToken(userID, role, t.RefreshSecret)
		if err != nil { return err }

		if err := SaveRefreshToken(tx, newRefresh, userID); err != nil {
			return err
		}
		return nil
	}); txErr != nil {
		return "", "", nil, txErr
	}

	return newAccess, newRefresh, claims, nil

}

func setUserContext(c echo.Context, claims jwt.MapClaims) {
	c.Set("userID", uint(claims["sub"].(float64)))
	c.Set("role", claims["role"].(string))
}


func ValidateRefresh(rawToken string, refreshsecret []byte, db *gorm.DB) (jwt.MapClaims, error) {
	t, err := jwt.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
            return nil, fmt.Errorf("unexpected alg: %s", t.Method.Alg())
        }
		return refreshsecret, nil
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
			return nil, errors.New("refresh token not found")
		}
		return nil, fmt.Errorf("db error: %w", err)
	}

	if stored.Revoked {
		return nil, fmt.Errorf("refresh token revoked")
	}
	if time.Now().Unix() > stored.ExpiresAt {
		return nil, fmt.Errorf("refresh token expired")
	}

	return claims, nil
}

func SignAccessToken(userID uint, role string, accessSecret []byte) (string, error) {
	exp := time.Now().Add(15 * time.Minute)
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  exp.Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(accessSecret)
}

func SignRefreshToken(userID uint, role string, refreshSecret []byte) (string, error) {
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
