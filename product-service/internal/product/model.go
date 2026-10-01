package product

import (
	"time"

	"gorm.io/gorm"

	"github.com/geoo115/E-commerceMicroservices/pkg/events"
)

// Category groups products.
type Category struct {
	ID        uint64 `gorm:"primaryKey"`
	Name      string `gorm:"size:64;uniqueIndex;not null"`
	CreatedAt time.Time
}

// Product is a catalog entry. Prices are stored in cents to avoid floating-point rounding.
type Product struct {
	ID          uint64   `gorm:"primaryKey"`
	Name        string   `gorm:"size:200;not null;index"`
	Description string   `gorm:"type:text"`
	PriceCents  int64    `gorm:"not null;check:price_cents > 0"`
	CategoryID  uint64   `gorm:"not null;index"`
	Category    Category `gorm:"constraint:OnDelete:RESTRICT"`
	// Rating aggregates are maintained from review-service events.
	RatingCount uint32 `gorm:"not null;default:0"`
	RatingSum   int64  `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// RatingAverage returns the mean review rating, or 0 without reviews.
func (p Product) RatingAverage() float64 {
	if p.RatingCount == 0 {
		return 0
	}
	return float64(p.RatingSum) / float64(p.RatingCount)
}

// Models lists the tables owned by product-service.
var Models = []any{&Category{}, &Product{}, &events.OutboxMessage{}, &events.ProcessedEvent{}}
