package handler

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
	cartv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/cart/v1"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	paymentv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/payment/v1"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
	reviewv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/review/v1"
)

// The gateway exposes its own JSON representations instead of serializing
// protobuf messages directly. The public API stays stable when internal
// contracts change, and enums are rendered as readable lowercase strings.

type addressDTO struct {
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2,omitempty"`
	City         string `json:"city"`
	State        string `json:"state,omitempty"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
}

type userDTO struct {
	ID            uint64      `json:"id"`
	Username      string      `json:"username"`
	Email         string      `json:"email"`
	Phone         string      `json:"phone,omitempty"`
	Role          string      `json:"role"`
	EmailVerified bool        `json:"email_verified"`
	Address       *addressDTO `json:"address,omitempty"`
}

func toUser(u *authv1.User) userDTO {
	out := userDTO{
		ID:            u.GetId(),
		Username:      u.GetUsername(),
		Email:         u.GetEmail(),
		Phone:         u.GetPhone(),
		Role:          u.GetRole(),
		EmailVerified: u.GetEmailVerified(),
	}
	if a := u.GetAddress(); a != nil {
		out.Address = &addressDTO{
			AddressLine1: a.GetAddressLine1(),
			AddressLine2: a.GetAddressLine2(),
			City:         a.GetCity(),
			State:        a.GetState(),
			PostalCode:   a.GetPostalCode(),
			Country:      a.GetCountry(),
		}
	}
	return out
}

type categoryDTO struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type stockDTO struct {
	ProductID uint64 `json:"product_id"`
	Available int32  `json:"available"`
	Reserved  int32  `json:"reserved"`
}

type productDTO struct {
	ID            uint64      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	PriceCents    int64       `json:"price_cents"`
	Category      categoryDTO `json:"category"`
	RatingAverage float64     `json:"rating_average"`
	RatingCount   uint32      `json:"rating_count"`
	// Available is composed from inventory-service; omitted if it is unreachable.
	Available *int32 `json:"available,omitempty"`
}

func toProduct(p *productv1.Product) productDTO {
	return productDTO{
		ID:            p.GetId(),
		Name:          p.GetName(),
		Description:   p.GetDescription(),
		PriceCents:    p.GetPriceCents(),
		Category:      categoryDTO{ID: p.GetCategory().GetId(), Name: p.GetCategory().GetName()},
		RatingAverage: p.GetRatingAverage(),
		RatingCount:   p.GetRatingCount(),
	}
}

type cartItemDTO struct {
	ProductID uint64 `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type cartDTO struct {
	Items []cartItemDTO `json:"items"`
}

func toCart(c *cartv1.Cart) cartDTO {
	out := cartDTO{Items: make([]cartItemDTO, 0, len(c.GetItems()))}
	for _, it := range c.GetItems() {
		out.Items = append(out.Items, cartItemDTO{ProductID: it.GetProductId(), Quantity: it.GetQuantity()})
	}
	return out
}

type orderItemDTO struct {
	ProductID      uint64 `json:"product_id"`
	Quantity       int32  `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

type orderDTO struct {
	ID                 uint64         `json:"id"`
	UserID             uint64         `json:"user_id"`
	Status             string         `json:"status"`
	TotalCents         int64          `json:"total_cents"`
	Items              []orderItemDTO `json:"items"`
	CancellationReason string         `json:"cancellation_reason,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func toOrder(o *orderv1.Order) orderDTO {
	out := orderDTO{
		ID:                 o.GetId(),
		UserID:             o.GetUserId(),
		Status:             enumName(o.GetStatus().String(), "ORDER_STATUS_"),
		TotalCents:         o.GetTotalCents(),
		Items:              make([]orderItemDTO, 0, len(o.GetItems())),
		CancellationReason: o.GetCancellationReason(),
		CreatedAt:          asTime(o.GetCreatedAt()),
		UpdatedAt:          asTime(o.GetUpdatedAt()),
	}
	for _, it := range o.GetItems() {
		out.Items = append(out.Items, orderItemDTO{
			ProductID:      it.GetProductId(),
			Quantity:       it.GetQuantity(),
			UnitPriceCents: it.GetUnitPriceCents(),
		})
	}
	return out
}

type paymentDTO struct {
	ID            uint64    `json:"id"`
	OrderID       uint64    `json:"order_id"`
	AmountCents   int64     `json:"amount_cents"`
	Currency      string    `json:"currency"`
	Method        string    `json:"method"`
	Status        string    `json:"status"`
	TransactionID string    `json:"transaction_id,omitempty"`
	FailureReason string    `json:"failure_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func toPayment(p *paymentv1.Payment) paymentDTO {
	return paymentDTO{
		ID:            p.GetId(),
		OrderID:       p.GetOrderId(),
		AmountCents:   p.GetAmountCents(),
		Currency:      p.GetCurrency(),
		Method:        p.GetMethod(),
		Status:        enumName(p.GetStatus().String(), "PAYMENT_STATUS_"),
		TransactionID: p.GetTransactionId(),
		FailureReason: p.GetFailureReason(),
		CreatedAt:     asTime(p.GetCreatedAt()),
	}
}

type reviewDTO struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"user_id"`
	ProductID uint64    `json:"product_id"`
	Rating    int32     `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

func toReview(r *reviewv1.Review) reviewDTO {
	return reviewDTO{
		ID:        r.GetId(),
		UserID:    r.GetUserId(),
		ProductID: r.GetProductId(),
		Rating:    r.GetRating(),
		Comment:   r.GetComment(),
		CreatedAt: asTime(r.GetCreatedAt()),
	}
}

type wishlistItemDTO struct {
	ProductID uint64    `json:"product_id"`
	AddedAt   time.Time `json:"added_at"`
}

// stockSource is implemented by both GetStockResponse and AdjustStockResponse.
type stockSource interface {
	GetProductId() uint64
	GetAvailable() int32
	GetReserved() int32
}

func toStock(s stockSource) stockDTO {
	return stockDTO{ProductID: s.GetProductId(), Available: s.GetAvailable(), Reserved: s.GetReserved()}
}

// page is the envelope of paginated responses.
type page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
}

func mapSlice[S, T any](in []S, f func(S) T) []T {
	out := make([]T, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func enumName(name, prefix string) string {
	return strings.ToLower(strings.TrimPrefix(name, prefix))
}

func asTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
