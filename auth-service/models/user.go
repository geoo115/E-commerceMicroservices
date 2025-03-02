// models/user.go
package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string    `gorm:"unique;not null"`
	Email    string    `gorm:"unique;not null"`
	Phone    string    `gorm:"unique;not null"`
	Password string    `gorm:"not null"`
	Role     string    `gorm:"default:'customer'"`
	IsActive bool      `gorm:"default:false"`
	Address  []Address `gorm:"foreignKey:UserID"`
}

type Address struct {
	gorm.Model
	UserID   uint   `json:"user_id"`
	Address  string `json:"address"`
	City     string `json:"city"`
	PostCode string `json:"post_code"`
}
