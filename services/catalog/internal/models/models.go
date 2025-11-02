package models

type Product struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"  json:"id" `
	Name        string  `gorm:"not null"                  json:"name"`
	Description string  `gorm:"not null"                  json:"description"`
	Price       float64 `gorm:"not null"                  json:"price"`
	Count       uint    `json:"count"`
}