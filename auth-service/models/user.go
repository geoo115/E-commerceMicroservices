package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string    `gorm:"unique;not null" json:"username"`
	Email    string    `gorm:"unique;not null" json:"email"`
	Phone    string    `gorm:"unique;not null" json:"phone"`
	Password string    `gorm:"not null" json:"-"`
	Role     string    `gorm:"default:'user'" json:"role"`
	Address  []Address `gorm:"foreignKey:UserID"`
}

type Address struct {
	gorm.Model
	UserID   uint   `json:"user_id"`
	Address  string `json:"address"`
	City     string `json:"city"`
	PostCode string `json:"post_code"`
}
