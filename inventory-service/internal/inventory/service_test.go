package inventory

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/geoo115/E-commerceMicroservices/pkg/database/dbtest"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	inventoryv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/inventory/v1"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestMergeItems_SumsAndSortsByProduct(t *testing.T) {
	got := mergeItems([]events.OrderItem{{ProductID: 3, Quantity: 1}, {ProductID: 1, Quantity: 2}, {ProductID: 3, Quantity: 4}})
	assert.Equal(t, []events.OrderItem{{ProductID: 1, Quantity: 2}, {ProductID: 3, Quantity: 5}}, got)
}

// --- integration tests (require TEST_DATABASE_URL) ---

func newService(t *testing.T) *Service {
	db := dbtest.New(t, Models...)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(20)
	return NewService(db, discard)
}

func seedStock(t *testing.T, s *Service, productID uint64, available int32) {
	require.NoError(t, s.db.Create(&Stock{ProductID: productID, Available: available}).Error)
}

func orderCreated(t *testing.T, orderID uint64, items ...events.OrderItem) events.Event {
	ev, err := events.New(events.OrderCreated, events.OrderCreatedData{OrderID: orderID, Items: items})
	require.NoError(t, err)
	return ev
}

func stockOf(t *testing.T, s *Service, productID uint64) Stock {
	var st Stock
	require.NoError(t, s.db.First(&st, "product_id = ?", productID).Error)
	return st
}

func outboxTypes(t *testing.T, s *Service) map[string]int {
	var msgs []events.OutboxMessage
	require.NoError(t, s.db.Find(&msgs).Error)
	out := map[string]int{}
	for _, m := range msgs {
		var ev events.Event
		require.NoError(t, json.Unmarshal(m.Payload, &ev))
		out[ev.Type]++
	}
	return out
}

// TestReserve_NoOversellUnderConcurrency fires 50 concurrent orders at a
// product with 10 units: exactly 10 must be reserved and 40 rejected.
func TestReserve_NoOversellUnderConcurrency(t *testing.T) {
	s := newService(t)
	seedStock(t, s, 1, 10)

	const orders = 50
	var wg sync.WaitGroup
	errs := make(chan error, orders)
	for i := 1; i <= orders; i++ {
		wg.Add(1)
		go func(orderID uint64) {
			defer wg.Done()
			errs <- s.HandleOrderEvent(context.Background(), orderCreated(t, orderID, events.OrderItem{ProductID: 1, Quantity: 1}))
		}(uint64(i))
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	st := stockOf(t, s, 1)
	assert.Equal(t, int32(0), st.Available)
	assert.Equal(t, int32(10), st.Reserved)
	assert.Equal(t, map[string]int{events.InventoryReserved: 10, events.InventoryRejected: 40}, outboxTypes(t, s))
}

// TestReserve_NoDeadlockWithOverlappingOrders reserves the same two products
// in opposite orders concurrently; sorted locking must prevent deadlocks.
func TestReserve_NoDeadlockWithOverlappingOrders(t *testing.T) {
	s := newService(t)
	seedStock(t, s, 1, 100)
	seedStock(t, s, 2, 100)

	var wg sync.WaitGroup
	for i := 1; i <= 40; i++ {
		wg.Add(1)
		go func(orderID uint64) {
			defer wg.Done()
			items := []events.OrderItem{{ProductID: 1, Quantity: 1}, {ProductID: 2, Quantity: 1}}
			if orderID%2 == 0 {
				items[0], items[1] = items[1], items[0]
			}
			assert.NoError(t, s.HandleOrderEvent(context.Background(), orderCreated(t, orderID, items...)))
		}(uint64(i))
	}
	wg.Wait()

	assert.Equal(t, int32(60), stockOf(t, s, 1).Available)
	assert.Equal(t, int32(60), stockOf(t, s, 2).Available)
}

func TestReserve_AllOrNothing(t *testing.T) {
	s := newService(t)
	seedStock(t, s, 1, 5)
	seedStock(t, s, 2, 1)

	err := s.HandleOrderEvent(context.Background(), orderCreated(t, 1,
		events.OrderItem{ProductID: 1, Quantity: 2}, events.OrderItem{ProductID: 2, Quantity: 3}))
	require.NoError(t, err)

	assert.Equal(t, int32(5), stockOf(t, s, 1).Available, "product 1 must not be reserved when product 2 is short")
	assert.Equal(t, map[string]int{events.InventoryRejected: 1}, outboxTypes(t, s))
}

func TestReserve_DuplicateDeliveryIsIdempotent(t *testing.T) {
	s := newService(t)
	seedStock(t, s, 1, 5)
	ev := orderCreated(t, 1, events.OrderItem{ProductID: 1, Quantity: 2})

	require.NoError(t, s.HandleOrderEvent(context.Background(), ev))
	require.NoError(t, s.HandleOrderEvent(context.Background(), ev))

	assert.Equal(t, int32(3), stockOf(t, s, 1).Available)
	assert.Equal(t, map[string]int{events.InventoryReserved: 1}, outboxTypes(t, s))
}

func TestCancelReleasesAndPaymentCommits(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	seedStock(t, s, 1, 10)
	require.NoError(t, s.HandleOrderEvent(ctx, orderCreated(t, 1, events.OrderItem{ProductID: 1, Quantity: 3})))
	require.NoError(t, s.HandleOrderEvent(ctx, orderCreated(t, 2, events.OrderItem{ProductID: 1, Quantity: 4})))

	cancelled, _ := events.New(events.OrderCancelled, events.OrderCancelledData{OrderID: 1})
	require.NoError(t, s.HandleOrderEvent(ctx, cancelled))
	paid, _ := events.New(events.PaymentSucceeded, events.PaymentResultData{OrderID: 2})
	require.NoError(t, s.HandlePaymentEvent(ctx, paid))

	st := stockOf(t, s, 1)
	assert.Equal(t, int32(6), st.Available, "3 released back, 4 sold")
	assert.Equal(t, int32(0), st.Reserved)
}

func TestAdjustStock(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	resp, err := s.AdjustStock(ctx, &inventoryv1.AdjustStockRequest{ProductId: 7, Delta: 5})
	require.NoError(t, err)
	assert.Equal(t, int32(5), resp.GetAvailable())

	_, err = s.AdjustStock(ctx, &inventoryv1.AdjustStockRequest{ProductId: 7, Delta: -6})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	_, err = s.AdjustStock(ctx, &inventoryv1.AdjustStockRequest{ProductId: 7})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
