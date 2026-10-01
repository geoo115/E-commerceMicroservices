package inventory

import (
	"time"

	"github.com/geoo115/E-commerceMicroservices/pkg/events"
)

// Stock holds the stock level of one product.
//
//	Available: units that can still be sold.
//	Reserved:  units held for orders that are not paid yet.
type Stock struct {
	ProductID uint64 `gorm:"primaryKey;autoIncrement:false"`
	Available int32  `gorm:"not null;check:available >= 0"`
	Reserved  int32  `gorm:"not null;default:0;check:reserved >= 0"`
	UpdatedAt time.Time
}

// Reservation statuses.
const (
	ReservationHeld      = "held"      // stock is held for an unpaid order
	ReservationCommitted = "committed" // order paid, units are sold
	ReservationReleased  = "released"  // order cancelled, units returned
)

// Reservation records the units held for one product of one order. The
// unique (order_id, product_id) index guarantees an order is reserved once.
type Reservation struct {
	ID        uint64 `gorm:"primaryKey"`
	OrderID   uint64 `gorm:"not null;uniqueIndex:idx_reservation_order_product"`
	ProductID uint64 `gorm:"not null;uniqueIndex:idx_reservation_order_product"`
	Quantity  int32  `gorm:"not null"`
	Status    string `gorm:"size:16;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Models lists the tables owned by inventory-service.
var Models = []any{&Stock{}, &Reservation{}, &events.OutboxMessage{}, &events.ProcessedEvent{}}
