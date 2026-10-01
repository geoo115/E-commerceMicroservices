package order

import (
	"time"

	"github.com/geoo115/E-commerceMicroservices/pkg/events"
)

// Order is a customer order. Unit prices are copied from the catalog when the
// order is placed, so later price changes do not alter existing orders.
type Order struct {
	ID                 uint64 `gorm:"primaryKey"`
	UserID             uint64 `gorm:"not null;index"`
	Status             Status `gorm:"size:16;not null;index"`
	TotalCents         int64  `gorm:"not null"`
	CancellationReason string
	Items              []Item `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Item is one product line of an order.
type Item struct {
	ID             uint64 `gorm:"primaryKey"`
	OrderID        uint64 `gorm:"not null;index"`
	ProductID      uint64 `gorm:"not null"`
	Quantity       int32  `gorm:"not null"`
	UnitPriceCents int64  `gorm:"not null"`
}

// TableName implements gorm's Tabler.
func (Item) TableName() string { return "order_items" }

// Models lists the tables owned by order-service.
var Models = []any{&Order{}, &Item{}, &events.OutboxMessage{}, &events.ProcessedEvent{}}
