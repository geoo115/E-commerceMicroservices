package models

import "gorm.io/gorm"

type Category struct {
	gorm.Model
	Name     string    `json:"name" gorm:"unique"`
	Products []Product `gorm:"foreignKey:CategoryID"`
}

type Product struct {
	gorm.Model
	Name        string    `json:"name"`
	Price       float64   `json:"price"`
	CategoryID  uint      `json:"category_id"`
	Description string    `json:"description"`
	Category    Category  `json:"category" gorm:"foreignKey:CategoryID"`
	Inventory   Inventory `json:"inventory" gorm:"foreignKey:ProductID;references:ID"`
	Reviews     []Review  `json:"reviews" gorm:"foreignKey:ProductID"`
}

type Inventory struct {
	gorm.Model
	ProductID uint     `json:"product_id"`
	Stock     int      `json:"stock"`
	Product   *Product `json:"-" gorm:"foreignKey:ProductID;references:ID"`
}

type Review struct {
	gorm.Model
	ProductID uint   `json:"product_id"`
	UserID    uint   `json:"user_id"`
	Rating    int    `json:"rating" gorm:"check:rating >= 1 AND rating <= 5"`
	Comment   string `json:"comment"`
}
