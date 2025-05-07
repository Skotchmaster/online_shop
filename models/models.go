package models

type Product struct {
	ID          int     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string `gorm:"unique;not null"      json:"username"`
	PasswordHash string `gorm:"not null"             json:"-"`
	Role         string `gorm:"not null;default:user" json:"role"`
}
