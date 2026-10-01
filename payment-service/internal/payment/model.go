package payment

import (
	"time"

	"gorm.io/gorm"

	"github.com/geoo115/E-commerceMicroservices/pkg/events"
)

// Payment statuses.
const (
	StatusPending   = "pending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

// Payment is one charge attempt for an order.
type Payment struct {
	ID            uint64 `gorm:"primaryKey"`
	OrderID       uint64 `gorm:"not null;index"`
	UserID        uint64 `gorm:"not null;index"`
	AmountCents   int64  `gorm:"not null;check:amount_cents > 0"`
	Currency      string `gorm:"size:3;not null"`
	Method        string `gorm:"size:16;not null"`
	CardLastFour  string `gorm:"size:4"`
	Status        string `gorm:"size:16;not null"`
	TransactionID string `gorm:"size:64"`
	FailureReason string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Migrate creates the payment tables. The partial unique index allows at most
// one pending-or-succeeded payment per order, which makes double charging
// impossible even when two requests race; failed attempts can be retried.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Payment{}, &events.OutboxMessage{}); err != nil {
		return err
	}
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_one_active_per_order
		ON payments (order_id) WHERE status IN ('pending', 'succeeded')`).Error
}
