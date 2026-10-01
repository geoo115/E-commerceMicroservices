package auth

import "time"

// Roles.
const (
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
)

// User is a registered account.
type User struct {
	ID            uint64   `gorm:"primaryKey"`
	Username      string   `gorm:"size:32;uniqueIndex;not null"`
	Email         string   `gorm:"size:254;uniqueIndex;not null"`
	Phone         string   `gorm:"size:32"`
	PasswordHash  string   `gorm:"not null"`
	Role          string   `gorm:"size:16;not null;default:customer"`
	EmailVerified bool     `gorm:"not null;default:false"`
	Address       *Address `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Address is the user's shipping address.
type Address struct {
	ID           uint64 `gorm:"primaryKey"`
	UserID       uint64 `gorm:"uniqueIndex;not null"`
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Country      string
}

// Models lists the tables owned by auth-service.
var Models = []any{&User{}, &Address{}}
