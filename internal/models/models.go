package models

import (
	"time"
)

type Product struct {
	ID          int     `gorm:"primaryKey;autoIncrement"  json:"id" `
	Name        string  `gorm:"not null"                  json:"name"`
	Description string  `gorm:"not null"                  json:"description"`
	Price       float64 `gorm:"not null"                  json:"price"`
	Count       uint    `json:"count"`
}

type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string `gorm:"unique;not null"          json:"username"`
	PasswordHash string `gorm:"not null"                 json:"-"`
	Role         string `gorm:"not null;"                json:"role"`
}

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey"          json:"id"`
	Token     string    `gorm:"unique;not null"     json:"token"`
	UserID    uint      `gorm:"index;not null"      json:"user_id"`
	ExpiresAt time.Time `gorm:"not null"            json:"expires_at"`
	Revoked   bool      `gorm:"default:false"       json:"revoked"`
}

type CartItem struct {
	ID        uint `gorm:"primaryKey"                  json:"id"`
	UserID    uint `gorm:"index;not null"              json:"user_id"`
	ProductID uint `gorm:"not null"                    json:"product_id"`
	Quantity  uint `gorm:"default:1;check:quantity>0"  json:"quantity"`
}
