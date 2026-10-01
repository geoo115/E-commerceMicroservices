// Package order implements order-service: the order lifecycle and the
// order-side steps of the choreography-based order saga.
package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/paging"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
)

const (
	consumerName    = "order-service"
	maxItems        = 50
	maxItemQuantity = 100
)

// Catalog is the subset of product-service used to price orders.
type Catalog interface {
	BatchGetProducts(ctx context.Context, in *productv1.BatchGetProductsRequest, opts ...grpc.CallOption) (*productv1.BatchGetProductsResponse, error)
}

// Service implements orderv1.OrderServiceServer and the saga event handlers.
type Service struct {
	orderv1.UnimplementedOrderServiceServer

	db      *gorm.DB
	catalog Catalog
	log     *slog.Logger
}

// NewService returns an order Service.
func NewService(db *gorm.DB, catalog Catalog, log *slog.Logger) *Service {
	return &Service{db: db, catalog: catalog, log: log}
}

// CreateOrder places an order in PENDING state and emits order.created.
// Prices always come from product-service: the request has no price field,
// and the order is rejected if the catalog cannot be reached.
func (s *Service) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	if req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	quantities, err := mergeItems(req.GetItems())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ids := make([]uint64, 0, len(quantities))
	for id := range quantities {
		ids = append(ids, id)
	}
	catalogCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := s.catalog.BatchGetProducts(catalogCtx, &productv1.BatchGetProductsRequest{Ids: ids})
	if status.Code(err) == codes.NotFound {
		return nil, status.Error(codes.NotFound, status.Convert(err).Message())
	}
	if err != nil {
		s.log.ErrorContext(ctx, "price lookup failed", "error", err)
		return nil, status.Error(codes.Unavailable, "product catalog is unavailable, try again later")
	}

	order := Order{UserID: req.GetUserId(), Status: StatusPending}
	for _, p := range resp.GetProducts() {
		q := quantities[p.GetId()]
		order.Items = append(order.Items, Item{ProductID: p.GetId(), Quantity: q, UnitPriceCents: p.GetPriceCents()})
		order.TotalCents += p.GetPriceCents() * int64(q)
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		data := events.OrderCreatedData{OrderID: order.ID, UserID: order.UserID, TotalCents: order.TotalCents}
		for _, it := range order.Items {
			data.Items = append(data.Items, events.OrderItem{ProductID: it.ProductID, Quantity: it.Quantity})
		}
		return enqueue(tx, order.ID, events.OrderCreated, data)
	})
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "create order", err)
	}
	return &orderv1.CreateOrderResponse{Order: toProto(order)}, nil
}

// GetOrder returns an order with its items.
func (s *Service) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	var o Order
	err := s.db.WithContext(ctx).Preload("Items").First(&o, req.GetOrderId()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "order not found")
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "get order", err)
	}
	return &orderv1.GetOrderResponse{Order: toProto(o)}, nil
}

// ListOrders returns a user's orders, newest first.
func (s *Service) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	q := s.db.WithContext(ctx).Model(&Order{}).Where("user_id = ?", req.GetUserId())

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "count orders", err)
	}
	limit, offset := paging.Normalize(req.GetPage(), req.GetPageSize())
	var orders []Order
	if err := q.Preload("Items").Order("id DESC").Limit(limit).Offset(offset).Find(&orders).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "list orders", err)
	}

	out := make([]*orderv1.Order, len(orders))
	for i, o := range orders {
		out[i] = toProto(o)
	}
	return &orderv1.ListOrdersResponse{Orders: out, Total: total}, nil
}

// CancelOrder cancels a pending or confirmed order and emits order.cancelled,
// which makes inventory-service release the reserved stock (compensation).
func (s *Service) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.CancelOrderResponse, error) {
	reason := req.GetReason()
	if reason == "" {
		reason = "cancelled by customer"
	}
	o, err := s.transition(ctx, req.GetOrderId(), StatusCancelled, reason)
	if err != nil {
		return nil, err
	}
	return &orderv1.CancelOrderResponse{Order: toProto(o)}, nil
}

// UpdateOrderStatus is used by back-office staff for fulfilment (shipped, delivered).
// Payment and stock states are driven by events and cannot be set manually.
func (s *Service) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	to, ok := StatusFromProto(req.GetStatus())
	if !ok || (to != StatusShipped && to != StatusDelivered) {
		return nil, status.Error(codes.InvalidArgument, "status must be SHIPPED or DELIVERED; use CancelOrder to cancel")
	}
	o, err := s.transition(ctx, req.GetOrderId(), to, "")
	if err != nil {
		return nil, err
	}
	return &orderv1.UpdateOrderStatusResponse{Order: toProto(o)}, nil
}

// HandleInventoryEvent confirms or cancels pending orders.
func (s *Service) HandleInventoryEvent(ctx context.Context, ev events.Event) error {
	switch ev.Type {
	case events.InventoryReserved:
		var data events.InventoryReservedData
		if err := ev.Decode(&data); err != nil {
			return err
		}
		return s.applyEvent(ctx, ev, data.OrderID, StatusConfirmed, "")
	case events.InventoryRejected:
		var data events.InventoryRejectedData
		if err := ev.Decode(&data); err != nil {
			return err
		}
		return s.applyEvent(ctx, ev, data.OrderID, StatusCancelled, data.Reason)
	}
	return nil
}

// HandlePaymentEvent marks confirmed orders as paid.
func (s *Service) HandlePaymentEvent(ctx context.Context, ev events.Event) error {
	if ev.Type != events.PaymentSucceeded {
		return nil // a failed payment leaves the order confirmed so the customer can retry
	}
	var data events.PaymentResultData
	if err := ev.Decode(&data); err != nil {
		return err
	}
	return s.applyEvent(ctx, ev, data.OrderID, StatusPaid, "")
}

// applyEvent performs a status transition requested by another service.
// Events that no longer apply (e.g. stock reserved for an order the customer
// already cancelled) are acknowledged and ignored.
func (s *Service) applyEvent(ctx context.Context, ev events.Event, orderID uint64, to Status, reason string) error {
	return events.ProcessOnce(ctx, s.db, consumerName, ev, func(tx *gorm.DB) error {
		_, err := transitionTx(tx, orderID, to, reason)
		if st, ok := status.FromError(err); ok && err != nil {
			s.log.WarnContext(ctx, "ignoring event", "event", ev.Type, "order_id", orderID, "reason", st.Message())
			return nil
		}
		return err
	})
}

func (s *Service) transition(ctx context.Context, orderID uint64, to Status, reason string) (Order, error) {
	var o Order
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		o, err = transitionTx(tx, orderID, to, reason)
		return err
	})
	if _, ok := status.FromError(err); ok && err != nil {
		return Order{}, err
	}
	if err != nil {
		return Order{}, grpcx.Internal(ctx, s.log, "update order status", err)
	}
	return o, nil
}

// transitionTx locks the order row, validates the transition against the
// state machine and emits order.cancelled when an order is cancelled.
func transitionTx(tx *gorm.DB, orderID uint64, to Status, reason string) (Order, error) {
	var o Order
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Items").First(&o, orderID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return o, status.Error(codes.NotFound, "order not found")
	}
	if err != nil {
		return o, err
	}
	if !CanTransition(o.Status, to) {
		return o, status.Errorf(codes.FailedPrecondition, "order is %s and cannot become %s", o.Status, to)
	}

	o.Status = to
	if to == StatusCancelled {
		o.CancellationReason = reason
	}
	if err := tx.Model(&o).Updates(map[string]any{"status": o.Status, "cancellation_reason": o.CancellationReason}).Error; err != nil {
		return o, err
	}
	if to == StatusCancelled {
		return o, enqueue(tx, o.ID, events.OrderCancelled, events.OrderCancelledData{OrderID: o.ID, Reason: reason})
	}
	return o, nil
}

func enqueue(tx *gorm.DB, orderID uint64, eventType string, data any) error {
	ev, err := events.New(eventType, data)
	if err != nil {
		return err
	}
	return events.Enqueue(tx, events.TopicOrders, strconv.FormatUint(orderID, 10), ev)
}

// mergeItems validates the requested items and sums quantities per product.
func mergeItems(items []*orderv1.CreateOrderRequest_Item) (map[uint64]int32, error) {
	if len(items) == 0 {
		return nil, errors.New("order must contain at least one item")
	}
	if len(items) > maxItems {
		return nil, fmt.Errorf("order may contain at most %d items", maxItems)
	}
	out := make(map[uint64]int32, len(items))
	for _, it := range items {
		if it.GetProductId() == 0 {
			return nil, errors.New("product_id is required")
		}
		if it.GetQuantity() < 1 {
			return nil, fmt.Errorf("quantity for product %d must be at least 1", it.GetProductId())
		}
		out[it.GetProductId()] += it.GetQuantity()
		if out[it.GetProductId()] > maxItemQuantity {
			return nil, fmt.Errorf("quantity for product %d must be at most %d", it.GetProductId(), maxItemQuantity)
		}
	}
	return out, nil
}

func toProto(o Order) *orderv1.Order {
	out := &orderv1.Order{
		Id:                 o.ID,
		UserId:             o.UserID,
		Status:             o.Status.Proto(),
		TotalCents:         o.TotalCents,
		CancellationReason: o.CancellationReason,
		CreatedAt:          timestamppb.New(o.CreatedAt),
		UpdatedAt:          timestamppb.New(o.UpdatedAt),
	}
	for _, it := range o.Items {
		out.Items = append(out.Items, &orderv1.OrderItem{
			ProductId:      it.ProductID,
			Quantity:       it.Quantity,
			UnitPriceCents: it.UnitPriceCents,
		})
	}
	return out
}
