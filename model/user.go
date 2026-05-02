package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email    string `gorm:"type:varchar(150);uniqueIndex;not null"`
	Password string `gorm:"type:varchar(255);not null"` // hash bcrypt
}
