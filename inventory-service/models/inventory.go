package models

import "gorm.io/gorm"

// Inventory holds the current stock for a product.
type Inventory struct {
	gorm.Model
	ProductID uint `json:"product_id" gorm:"uniqueIndex"`
	Stock     int  `json:"stock" gorm:"check:stock >= 0"`
}

// Product represents a product with pricing and inventory details.
type Product struct {
	gorm.Model
	Name       string    `json:"name"`
	Price      float64   `json:"price"`
	CategoryID uint      `json:"category_id"`
	Category   Category  `json:"-" gorm:"foreignKey:CategoryID;constraint:OnDelete:CASCADE;"`
	Inventory  Inventory `json:"-" gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE;"`
}

// Category represents a product category.
type Category struct {
	gorm.Model
	Name     string    `json:"name" gorm:"unique"`
	Products []Product `json:"-" gorm:"foreignKey:CategoryID"`
}

// Order represents an order record.
type Order struct {
	gorm.Model
	Items []OrderItem `json:"-" gorm:"foreignKey:OrderID"`
}

// OrderItem represents an item in an order.
type OrderItem struct {
	gorm.Model
	OrderID   uint    `json:"order_id"`
	ProductID uint    `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}
