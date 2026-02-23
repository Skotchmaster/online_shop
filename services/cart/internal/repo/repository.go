package repo

import "gorm.io/gorm"

type GormRepo struct {
	DB            *gorm.DB
}