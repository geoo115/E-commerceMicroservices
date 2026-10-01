package events

// Topics. Each aggregate has one topic and messages are keyed by the aggregate
// ID, so all events about the same order (created, cancelled, ...) are
// delivered in order to every consumer group.
const (
	TopicOrders    = "orders"
	TopicInventory = "inventory"
	TopicPayments  = "payments"
	TopicProducts  = "products"
	TopicReviews   = "reviews"
)

// AllTopics lists every topic used by the platform.
var AllTopics = []string{TopicOrders, TopicInventory, TopicPayments, TopicProducts, TopicReviews}

// Event types.
const (
	OrderCreated      = "order.created"
	OrderCancelled    = "order.cancelled"
	InventoryReserved = "inventory.reserved"
	InventoryRejected = "inventory.rejected"
	PaymentSucceeded  = "payment.succeeded"
	PaymentFailed     = "payment.failed"
	ProductCreated    = "product.created"
	ReviewCreated     = "review.created"
	ReviewDeleted     = "review.deleted"
)

// OrderItem is a product line inside order events.
type OrderItem struct {
	ProductID uint64 `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

// OrderCreatedData is published by order-service when an order is placed.
type OrderCreatedData struct {
	OrderID    uint64      `json:"order_id"`
	UserID     uint64      `json:"user_id"`
	TotalCents int64       `json:"total_cents"`
	Items      []OrderItem `json:"items"`
}

// OrderCancelledData is published by order-service when an order is cancelled.
type OrderCancelledData struct {
	OrderID uint64 `json:"order_id"`
	Reason  string `json:"reason"`
}

// InventoryReservedData is published by inventory-service once stock for every item is reserved.
type InventoryReservedData struct {
	OrderID uint64 `json:"order_id"`
}

// InventoryRejectedData is published by inventory-service when an order cannot be fulfilled.
type InventoryRejectedData struct {
	OrderID uint64 `json:"order_id"`
	Reason  string `json:"reason"`
}

// PaymentResultData is published by payment-service for payment.succeeded and payment.failed.
type PaymentResultData struct {
	PaymentID   uint64 `json:"payment_id"`
	OrderID     uint64 `json:"order_id"`
	AmountCents int64  `json:"amount_cents"`
	Reason      string `json:"reason,omitempty"`
}

// ProductCreatedData is published by product-service when a product is added to the catalog.
type ProductCreatedData struct {
	ProductID    uint64 `json:"product_id"`
	InitialStock int32  `json:"initial_stock"`
}

// ReviewData is published by review-service for review.created and review.deleted.
type ReviewData struct {
	ReviewID  uint64 `json:"review_id"`
	ProductID uint64 `json:"product_id"`
	Rating    int32  `json:"rating"`
}
