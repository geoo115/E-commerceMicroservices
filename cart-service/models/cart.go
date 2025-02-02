// cart-service/models/cart.go
package models

import "gorm.io/gorm"

type Cart struct {
	gorm.Model
	UserID    uint `json:"user_id"`
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}
