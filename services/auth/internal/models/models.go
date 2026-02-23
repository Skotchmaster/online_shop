package models

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Username     string    `json:"username" gorm:"type:text;not null;uniqueIndex"`
	PasswordHash string    `json:"-" gorm:"type:text;not null"`
	Role         string    `json:"role" gorm:"type:text;not null"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Token     string    `json:"token_hash" gorm:"type:text;not null;index:idx_refresh_token_hash"`
	JTI       string    `json:"jti" gorm:"type:text;not null;uniqueIndex"`
	ExpiresAt time.Time `json:"expires_at" gorm:"type:timestamptz;not null;index:idx_refresh_expires_at"`
	Revoked   bool      `json:"revoked" gorm:"not null;default:false;index"`
}
