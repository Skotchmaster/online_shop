package models

type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string `gorm:"unique;not null"          json:"username"`
	PasswordHash string `gorm:"not null"                 json:"-"`
	Role         string `gorm:"not null;"                json:"role"`
}

type RefreshToken struct {
	ID        uint   `gorm:"primaryKey"            json:"id"`
	Role      string `gorm:"not null"              json:"role"`
	Token     string `gorm:"unique;not null"       json:"token"`
	UserID    uint   `gorm:"index;not null"        json:"user_id"`
	JTI       string `gorm:"not null, uniqueIndex" json:"jti"`
	ExpiresAt int64  `gorm:"not null"              json:"expires_at"`
	Revoked   bool   `gorm:"default:false"         json:"revoked"`
}