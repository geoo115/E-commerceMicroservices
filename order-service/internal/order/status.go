package order

import orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"

// Status is the lifecycle state of an order.
type Status string

// Order statuses.
const (
	StatusPending   Status = "pending"   // waiting for inventory to reserve stock
	StatusConfirmed Status = "confirmed" // stock reserved, waiting for payment
	StatusPaid      Status = "paid"
	StatusShipped   Status = "shipped"
	StatusDelivered Status = "delivered"
	StatusCancelled Status = "cancelled"
)

// transitions is the order state machine. Anything not listed is rejected.
//
//	pending ──► confirmed ──► paid ──► shipped ──► delivered
//	   │            │
//	   └──► cancelled ◄──┘
var transitions = map[Status][]Status{
	StatusPending:   {StatusConfirmed, StatusCancelled},
	StatusConfirmed: {StatusPaid, StatusCancelled},
	StatusPaid:      {StatusShipped},
	StatusShipped:   {StatusDelivered},
}

// CanTransition reports whether an order may move from one status to another.
func CanTransition(from, to Status) bool {
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

var toProtoStatus = map[Status]orderv1.OrderStatus{
	StatusPending:   orderv1.OrderStatus_ORDER_STATUS_PENDING,
	StatusConfirmed: orderv1.OrderStatus_ORDER_STATUS_CONFIRMED,
	StatusPaid:      orderv1.OrderStatus_ORDER_STATUS_PAID,
	StatusShipped:   orderv1.OrderStatus_ORDER_STATUS_SHIPPED,
	StatusDelivered: orderv1.OrderStatus_ORDER_STATUS_DELIVERED,
	StatusCancelled: orderv1.OrderStatus_ORDER_STATUS_CANCELLED,
}

// Proto converts the status to its protobuf enum.
func (s Status) Proto() orderv1.OrderStatus { return toProtoStatus[s] }

// StatusFromProto converts a protobuf enum to a Status.
func StatusFromProto(p orderv1.OrderStatus) (Status, bool) {
	for s, ps := range toProtoStatus {
		if ps == p {
			return s, true
		}
	}
	return "", false
}
