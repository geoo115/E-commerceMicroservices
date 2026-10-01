// Package payment implements payment-service.
package payment

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	paymentv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/payment/v1"
)

var (
	validMethods = map[string]bool{"card": true, "paypal": true, "wallet": true}
	lastFour     = regexp.MustCompile(`^[0-9]{4}$`)
)

// Orders is the subset of order-service used to load the order being paid.
type Orders interface {
	GetOrder(ctx context.Context, in *orderv1.GetOrderRequest, opts ...grpc.CallOption) (*orderv1.GetOrderResponse, error)
}

// Service implements paymentv1.PaymentServiceServer.
type Service struct {
	paymentv1.UnimplementedPaymentServiceServer

	db       *gorm.DB
	orders   Orders
	provider Provider
	currency string
	log      *slog.Logger
}

// NewService returns a payment Service.
func NewService(db *gorm.DB, orders Orders, provider Provider, currency string, log *slog.Logger) *Service {
	return &Service{db: db, orders: orders, provider: provider, currency: currency, log: log}
}

// ProcessPayment charges the total of a confirmed order owned by the caller.
// A declined charge is not an RPC error: it returns a FAILED payment.
func (s *Service) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.ProcessPaymentResponse, error) {
	if !validMethods[req.GetMethod()] {
		return nil, status.Error(codes.InvalidArgument, "method must be one of: card, paypal, wallet")
	}
	if req.GetMethod() == "card" && !lastFour.MatchString(req.GetCardLastFour()) {
		return nil, status.Error(codes.InvalidArgument, "card_last_four must be 4 digits")
	}

	order, err := s.loadPayableOrder(ctx, req.GetOrderId(), req.GetUserId())
	if err != nil {
		return nil, err
	}

	// 1. Claim the order with a pending payment. The partial unique index
	//    rejects a second concurrent attempt before anything is charged.
	p := Payment{
		OrderID:      order.GetId(),
		UserID:       order.GetUserId(),
		AmountCents:  order.GetTotalCents(), // the amount always comes from the order
		Currency:     s.currency,
		Method:       req.GetMethod(),
		CardLastFour: req.GetCardLastFour(),
		Status:       StatusPending,
	}
	if err := s.db.WithContext(ctx).Create(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, status.Error(codes.AlreadyExists, "this order is already paid or a payment is in progress")
		}
		return nil, grpcx.Internal(ctx, s.log, "create payment", err)
	}

	// 2. Charge the provider.
	txID, chargeErr := s.provider.Charge(ctx, p)
	if chargeErr != nil && !errors.Is(chargeErr, ErrDeclined) {
		s.log.ErrorContext(ctx, "payment provider error", "payment_id", p.ID, "error", chargeErr)
	}

	// 3. Record the outcome and its event atomically.
	eventType := events.PaymentSucceeded
	p.Status, p.TransactionID = StatusSucceeded, txID
	if chargeErr != nil {
		eventType = events.PaymentFailed
		p.Status, p.FailureReason = StatusFailed, chargeErr.Error()
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		ev, err := events.New(eventType, events.PaymentResultData{
			PaymentID:   p.ID,
			OrderID:     p.OrderID,
			AmountCents: p.AmountCents,
			Reason:      p.FailureReason,
		})
		if err != nil {
			return err
		}
		return events.Enqueue(tx, events.TopicPayments, strconv.FormatUint(p.OrderID, 10), ev)
	})
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "record payment result", err)
	}
	return &paymentv1.ProcessPaymentResponse{Payment: toProto(p)}, nil
}

// GetPaymentByOrder returns the most recent payment attempt for an order.
func (s *Service) GetPaymentByOrder(ctx context.Context, req *paymentv1.GetPaymentByOrderRequest) (*paymentv1.GetPaymentByOrderResponse, error) {
	var p Payment
	err := s.db.WithContext(ctx).Where("order_id = ?", req.GetOrderId()).Order("id DESC").First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "no payment for this order")
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "get payment", err)
	}
	return &paymentv1.GetPaymentByOrderResponse{Payment: toProto(p)}, nil
}

func (s *Service) loadPayableOrder(ctx context.Context, orderID, userID uint64) (*orderv1.Order, error) {
	orderCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := s.orders.GetOrder(orderCtx, &orderv1.GetOrderRequest{OrderId: orderID})
	if status.Code(err) == codes.NotFound {
		return nil, status.Error(codes.NotFound, "order not found")
	}
	if err != nil {
		s.log.ErrorContext(ctx, "order lookup failed", "error", err)
		return nil, status.Error(codes.Unavailable, "order service is unavailable, try again later")
	}

	order := resp.GetOrder()
	if order.GetUserId() != userID {
		return nil, status.Error(codes.PermissionDenied, "order belongs to another user")
	}
	switch order.GetStatus() {
	case orderv1.OrderStatus_ORDER_STATUS_CONFIRMED:
		return order, nil
	case orderv1.OrderStatus_ORDER_STATUS_PENDING:
		return nil, status.Error(codes.FailedPrecondition, "order is still being confirmed, retry shortly")
	case orderv1.OrderStatus_ORDER_STATUS_PAID, orderv1.OrderStatus_ORDER_STATUS_SHIPPED, orderv1.OrderStatus_ORDER_STATUS_DELIVERED:
		return nil, status.Error(codes.AlreadyExists, "order is already paid")
	default:
		return nil, status.Error(codes.FailedPrecondition, "order cannot be paid")
	}
}

func toProto(p Payment) *paymentv1.Payment {
	st := paymentv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	switch p.Status {
	case StatusSucceeded:
		st = paymentv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED
	case StatusFailed:
		st = paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED
	}
	return &paymentv1.Payment{
		Id:            p.ID,
		OrderId:       p.OrderID,
		UserId:        p.UserID,
		AmountCents:   p.AmountCents,
		Currency:      p.Currency,
		Method:        p.Method,
		Status:        st,
		TransactionId: p.TransactionID,
		FailureReason: p.FailureReason,
		CreatedAt:     timestamppb.New(p.CreatedAt),
	}
}
