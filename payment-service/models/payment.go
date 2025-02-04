// payment-service/models/payment.go
package models

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PaymentStatus represents possible payment states
type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "Pending"
	PaymentSuccess   PaymentStatus = "Success"
	PaymentFailed    PaymentStatus = "Failed"
	PaymentRefunded  PaymentStatus = "Refunded"
	PaymentCancelled PaymentStatus = "Cancelled"
)

// PaymentMethod represents supported payment methods
type PaymentMethod string

const (
	CreditCard     PaymentMethod = "Credit Card"
	PayPal         PaymentMethod = "PayPal"
	BankTransfer   PaymentMethod = "Bank Transfer"
	CashOnDelivery PaymentMethod = "Cash on Delivery"
	DigitalWallet  PaymentMethod = "Digital Wallet"
)

// Payment represents a payment transaction
type Payment struct {
	gorm.Model
	OrderID         uint          `json:"order_id" gorm:"index;not null"`
	TransactionID   string        `json:"transaction_id" gorm:"uniqueIndex;size:255"`
	PaymentMethod   PaymentMethod `json:"payment_method" gorm:"type:varchar(50);not null"`
	Amount          float64       `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency        string        `json:"currency" gorm:"type:varchar(3);default:'USD'"`
	Status          PaymentStatus `json:"status" gorm:"type:varchar(20);index;not null"`
	ProcessedAt     time.Time     `json:"processed_at"`
	FailureReason   string        `json:"failure_reason" gorm:"type:text"`
	CardLastFour    string        `json:"card_last_four" gorm:"type:varchar(4)"`
	PaymentGateway  string        `json:"payment_gateway" gorm:"type:varchar(50)"`
	GatewayResponse string        `json:"gateway_response" gorm:"type:text"`
}

// Validate checks payment constraints before saving
func (p *Payment) Validate() error {
	if p.Amount <= 0 {
		return errors.New("payment amount must be positive")
	}

	if !IsValidPaymentMethod(p.PaymentMethod) {
		return fmt.Errorf("invalid payment method: %s", p.PaymentMethod)
	}

	if !IsValidPaymentStatus(p.Status) {
		return fmt.Errorf("invalid payment status: %s", p.Status)
	}

	return nil
}

// BeforeSave GORM hook for validation
func (p *Payment) BeforeSave(tx *gorm.DB) error {
	return p.Validate()
}

// IsValidPaymentStatus checks if a status is valid
func IsValidPaymentStatus(status PaymentStatus) bool {
	switch status {
	case PaymentPending, PaymentSuccess, PaymentFailed, PaymentRefunded, PaymentCancelled:
		return true
	default:
		return false
	}
}

// IsValidPaymentMethod checks if a payment method is valid
func IsValidPaymentMethod(method PaymentMethod) bool {
	switch method {
	case CreditCard, PayPal, BankTransfer, CashOnDelivery, DigitalWallet:
		return true
	default:
		return false
	}
}

// GetValidPaymentStatuses returns list of valid statuses
func GetValidPaymentStatuses() []PaymentStatus {
	return []PaymentStatus{
		PaymentPending,
		PaymentSuccess,
		PaymentFailed,
		PaymentRefunded,
		PaymentCancelled,
	}
}

// GetValidPaymentMethods returns list of valid payment methods
func GetValidPaymentMethods() []PaymentMethod {
	return []PaymentMethod{
		CreditCard,
		PayPal,
		BankTransfer,
		CashOnDelivery,
		DigitalWallet,
	}
}

// String representation for logging
func (p Payment) String() string {
	return fmt.Sprintf("Payment{ID: %d, OrderID: %d, Amount: %.2f, Status: %s}",
		p.ID, p.OrderID, p.Amount, p.Status)
}
