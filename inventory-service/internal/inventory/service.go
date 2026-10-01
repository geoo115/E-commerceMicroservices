// Package inventory implements inventory-service: stock levels and the
// stock-reservation step of the order saga.
package inventory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	inventoryv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/inventory/v1"
)

// consumerName identifies this service in the processed_events table.
const consumerName = "inventory-service"

// Service implements inventoryv1.InventoryServiceServer and the event handlers.
type Service struct {
	inventoryv1.UnimplementedInventoryServiceServer

	db  *gorm.DB
	log *slog.Logger
}

// NewService returns an inventory Service.
func NewService(db *gorm.DB, log *slog.Logger) *Service {
	return &Service{db: db, log: log}
}

// GetStock returns the stock of a product.
func (s *Service) GetStock(ctx context.Context, req *inventoryv1.GetStockRequest) (*inventoryv1.GetStockResponse, error) {
	var st Stock
	err := s.db.WithContext(ctx).First(&st, "product_id = ?", req.GetProductId()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "no stock record for product")
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "get stock", err)
	}
	return &inventoryv1.GetStockResponse{ProductId: st.ProductID, Available: st.Available, Reserved: st.Reserved}, nil
}

// AdjustStock adds (restock) or removes (write-off) available units.
// The row is locked with SELECT ... FOR UPDATE so concurrent adjustments and
// reservations serialize instead of overwriting each other.
func (s *Service) AdjustStock(ctx context.Context, req *inventoryv1.AdjustStockRequest) (*inventoryv1.AdjustStockResponse, error) {
	if req.GetDelta() == 0 {
		return nil, status.Error(codes.InvalidArgument, "delta must not be zero")
	}

	var st Stock
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&st, "product_id = ?", req.GetProductId()).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			st = Stock{ProductID: req.GetProductId()}
		} else if err != nil {
			return err
		}

		if st.Available+req.GetDelta() < 0 {
			return status.Errorf(codes.FailedPrecondition, "only %d units available", st.Available)
		}
		st.Available += req.GetDelta()
		return tx.Save(&st).Error
	})
	if _, ok := status.FromError(err); ok && err != nil {
		return nil, err
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "adjust stock", err)
	}
	return &inventoryv1.AdjustStockResponse{ProductId: st.ProductID, Available: st.Available, Reserved: st.Reserved}, nil
}

// HandleProductEvent creates the stock record of a new product.
func (s *Service) HandleProductEvent(ctx context.Context, ev events.Event) error {
	if ev.Type != events.ProductCreated {
		return nil
	}
	var data events.ProductCreatedData
	if err := ev.Decode(&data); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&Stock{ProductID: data.ProductID, Available: data.InitialStock}).Error
}

// HandleOrderEvent reserves stock for new orders and releases it for cancelled ones.
func (s *Service) HandleOrderEvent(ctx context.Context, ev events.Event) error {
	switch ev.Type {
	case events.OrderCreated:
		var data events.OrderCreatedData
		if err := ev.Decode(&data); err != nil {
			return err
		}
		return events.ProcessOnce(ctx, s.db, consumerName, ev, func(tx *gorm.DB) error {
			return reserve(tx, data)
		})
	case events.OrderCancelled:
		var data events.OrderCancelledData
		if err := ev.Decode(&data); err != nil {
			return err
		}
		return events.ProcessOnce(ctx, s.db, consumerName, ev, func(tx *gorm.DB) error {
			return settle(tx, data.OrderID, ReservationReleased)
		})
	}
	return nil
}

// HandlePaymentEvent turns held units into sold units once an order is paid.
func (s *Service) HandlePaymentEvent(ctx context.Context, ev events.Event) error {
	if ev.Type != events.PaymentSucceeded {
		return nil
	}
	var data events.PaymentResultData
	if err := ev.Decode(&data); err != nil {
		return err
	}
	return events.ProcessOnce(ctx, s.db, consumerName, ev, func(tx *gorm.DB) error {
		return settle(tx, data.OrderID, ReservationCommitted)
	})
}

// reserve holds stock for every item of an order, or for none of them.
// Rows are locked in product_id order so that two orders containing the same
// products can never deadlock. It emits inventory.reserved or inventory.rejected.
func reserve(tx *gorm.DB, order events.OrderCreatedData) error {
	items := mergeItems(order.Items)
	key := strconv.FormatUint(order.OrderID, 10)

	stocks := make([]Stock, len(items))
	for i, it := range items {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&stocks[i], "product_id = ?", it.ProductID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return reject(tx, key, order.OrderID, fmt.Sprintf("product %d is not stocked", it.ProductID))
		}
		if err != nil {
			return err
		}
		if stocks[i].Available < it.Quantity {
			return reject(tx, key, order.OrderID, fmt.Sprintf("insufficient stock for product %d", it.ProductID))
		}
	}

	for i, it := range items {
		stocks[i].Available -= it.Quantity
		stocks[i].Reserved += it.Quantity
		if err := tx.Save(&stocks[i]).Error; err != nil {
			return err
		}
		res := Reservation{OrderID: order.OrderID, ProductID: it.ProductID, Quantity: it.Quantity, Status: ReservationHeld}
		if err := tx.Create(&res).Error; err != nil {
			return err
		}
	}

	ev, err := events.New(events.InventoryReserved, events.InventoryReservedData{OrderID: order.OrderID})
	if err != nil {
		return err
	}
	return events.Enqueue(tx, events.TopicInventory, key, ev)
}

func reject(tx *gorm.DB, key string, orderID uint64, reason string) error {
	ev, err := events.New(events.InventoryRejected, events.InventoryRejectedData{OrderID: orderID, Reason: reason})
	if err != nil {
		return err
	}
	return events.Enqueue(tx, events.TopicInventory, key, ev)
}

// settle moves the held reservations of an order to the given final status:
// released units go back to Available, committed units leave the stock.
func settle(tx *gorm.DB, orderID uint64, to string) error {
	var held []Reservation
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ? AND status = ?", orderID, ReservationHeld).
		Order("product_id").
		Find(&held).Error; err != nil {
		return err
	}

	for _, r := range held {
		var st Stock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&st, "product_id = ?", r.ProductID).Error; err != nil {
			return err
		}
		st.Reserved -= r.Quantity
		if to == ReservationReleased {
			st.Available += r.Quantity
		}
		if err := tx.Save(&st).Error; err != nil {
			return err
		}
		if err := tx.Model(&r).Update("status", to).Error; err != nil {
			return err
		}
	}
	return nil
}

// mergeItems sums quantities per product and sorts by product ID.
func mergeItems(items []events.OrderItem) []events.OrderItem {
	byProduct := make(map[uint64]int32, len(items))
	for _, it := range items {
		byProduct[it.ProductID] += it.Quantity
	}
	out := make([]events.OrderItem, 0, len(byProduct))
	for id, q := range byProduct {
		out = append(out, events.OrderItem{ProductID: id, Quantity: q})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductID < out[j].ProductID })
	return out
}
