// review-service/models/review.go
package models

import "gorm.io/gorm"

type Review struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index"`
	ProductID uint   `json:"product_id" gorm:"index"`
	Rating    int    `json:"rating" gorm:"check:rating >= 1 AND rating <= 5"`
	Comment   string `json:"comment"`
}

type Wishlist struct {
	gorm.Model
	UserID    uint `json:"user_id" gorm:"index"`
	ProductID uint `json:"product_id" gorm:"index"`

	// Composite index to prevent duplicates
	UniqueIndex string `gorm:"uniqueIndex:idx_user_product"`
}
