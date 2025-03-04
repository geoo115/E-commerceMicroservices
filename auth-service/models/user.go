// models/user.go
package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string  `gorm:"unique;not null"`
	Email    string  `gorm:"unique;not null"`
	Phone    string  `gorm:"unique;not null"`
	Password string  `gorm:"not null"`
	Role     string  `gorm:"default:'customer'"`
	IsActive bool    `gorm:"default:true"`
	Address  Address `gorm:"foreignKey:UserID"`
}

type Address struct {
	gorm.Model
	UserID       uint   `json:"user_id"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
}
