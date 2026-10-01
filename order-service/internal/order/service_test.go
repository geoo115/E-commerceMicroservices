package order

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/geoo115/E-commerceMicroservices/pkg/database/dbtest"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

type fakeCatalog struct {
	prices map[uint64]int64
	err    error
}

func (f fakeCatalog) BatchGetProducts(_ context.Context, in *productv1.BatchGetProductsRequest, _ ...grpc.CallOption) (*productv1.BatchGetProductsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := &productv1.BatchGetProductsResponse{}
	for _, id := range in.GetIds() {
		price, ok := f.prices[id]
		if !ok {
			return nil, status.Errorf(codes.NotFound, "product %d not found", id)
		}
		out.Products = append(out.Products, &productv1.Product{Id: id, PriceCents: price})
	}
	return out, nil
}

func items(pairs ...uint64) []*orderv1.CreateOrderRequest_Item {
	var out []*orderv1.CreateOrderRequest_Item
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, &orderv1.CreateOrderRequest_Item{ProductId: pairs[i], Quantity: int32(pairs[i+1])})
	}
	return out
}

func TestMergeItems(t *testing.T) {
	got, err := mergeItems(items(1, 2, 2, 1, 1, 3))
	require.NoError(t, err)
	assert.Equal(t, map[uint64]int32{1: 5, 2: 1}, got)

	for name, in := range map[string][]*orderv1.CreateOrderRequest_Item{
		"empty":          nil,
		"zero quantity":  items(1, 0),
		"missing id":     items(0, 1),
		"merged too big": items(1, 60, 1, 60),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := mergeItems(in)
			assert.Error(t, err)
		})
	}
}

// These paths fail before touching the database, so no DB is needed.
func TestCreateOrder_CatalogErrors(t *testing.T) {
	tests := []struct {
		name    string
		catalog fakeCatalog
		want    codes.Code
	}{
		{"unknown product", fakeCatalog{prices: map[uint64]int64{}}, codes.NotFound},
		{"catalog down: never fall back to client prices", fakeCatalog{err: status.Error(codes.Unavailable, "down")}, codes.Unavailable},
		{"catalog timeout", fakeCatalog{err: status.Error(codes.DeadlineExceeded, "slow")}, codes.Unavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(nil, tt.catalog, discard)
			_, err := svc.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{UserId: 1, Items: items(7, 1)})
			assert.Equal(t, tt.want, status.Code(err))
		})
	}
}

func TestCreateOrder_Validation(t *testing.T) {
	svc := NewService(nil, fakeCatalog{}, discard)
	_, err := svc.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{Items: items(1, 1)})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "missing user")
	_, err = svc.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{UserId: 1})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "no items")
}

// ---------------------------------------------------------------------------
// Integration tests (require TEST_DATABASE_URL)
// ---------------------------------------------------------------------------

func newIntegrationService(t *testing.T, catalog Catalog) *Service {
	return NewService(dbtest.New(t, Models...), catalog, discard)
}

func TestCreateOrder_PricesFromCatalogAndWritesOutbox(t *testing.T) {
	svc := newIntegrationService(t, fakeCatalog{prices: map[uint64]int64{1: 1250, 2: 999}})

	resp, err := svc.CreateOrder(context.Background(), &orderv1.CreateOrderRequest{UserId: 9, Items: items(1, 2, 2, 1, 1, 1)})
	require.NoError(t, err)
	o := resp.GetOrder()
	assert.Equal(t, orderv1.OrderStatus_ORDER_STATUS_PENDING, o.GetStatus())
	assert.Equal(t, int64(3*1250+999), o.GetTotalCents())
	assert.Len(t, o.GetItems(), 2)

	var outbox []events.OutboxMessage
	require.NoError(t, svc.db.Find(&outbox).Error)
	require.Len(t, outbox, 1, "order.created must be stored in the same transaction")
	assert.Equal(t, events.TopicOrders, outbox[0].Topic)
}

func TestSaga_OrderLifecycle(t *testing.T) {
	ctx := context.Background()
	svc := newIntegrationService(t, fakeCatalog{prices: map[uint64]int64{1: 100}})
	created, err := svc.CreateOrder(ctx, &orderv1.CreateOrderRequest{UserId: 1, Items: items(1, 1)})
	require.NoError(t, err)
	id := created.GetOrder().GetId()

	statusOf := func() Status {
		var o Order
		require.NoError(t, svc.db.First(&o, id).Error)
		return o.Status
	}
	event := func(typ string, data any) events.Event {
		ev, err := events.New(typ, data)
		require.NoError(t, err)
		return ev
	}

	reserved := event(events.InventoryReserved, events.InventoryReservedData{OrderID: id})
	require.NoError(t, svc.HandleInventoryEvent(ctx, reserved))
	assert.Equal(t, StatusConfirmed, statusOf())

	// Redelivery of the same event is a no-op.
	require.NoError(t, svc.HandleInventoryEvent(ctx, reserved))
	assert.Equal(t, StatusConfirmed, statusOf())

	require.NoError(t, svc.HandlePaymentEvent(ctx, event(events.PaymentSucceeded, events.PaymentResultData{OrderID: id})))
	assert.Equal(t, StatusPaid, statusOf())

	// A paid order cannot be cancelled.
	_, err = svc.CancelOrder(ctx, &orderv1.CancelOrderRequest{OrderId: id})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	// Stale events that no longer apply are acknowledged and ignored.
	require.NoError(t, svc.HandleInventoryEvent(ctx, event(events.InventoryRejected, events.InventoryRejectedData{OrderID: id})))
	assert.Equal(t, StatusPaid, statusOf())
}

func TestCancelOrder_EmitsCompensationEvent(t *testing.T) {
	ctx := context.Background()
	svc := newIntegrationService(t, fakeCatalog{prices: map[uint64]int64{1: 100}})
	created, err := svc.CreateOrder(ctx, &orderv1.CreateOrderRequest{UserId: 1, Items: items(1, 1)})
	require.NoError(t, err)

	resp, err := svc.CancelOrder(ctx, &orderv1.CancelOrderRequest{OrderId: created.GetOrder().GetId(), Reason: "test"})
	require.NoError(t, err)
	assert.Equal(t, orderv1.OrderStatus_ORDER_STATUS_CANCELLED, resp.GetOrder().GetStatus())

	var count int64
	require.NoError(t, svc.db.Model(&events.OutboxMessage{}).Count(&count).Error)
	assert.Equal(t, int64(2), count, "order.created + order.cancelled")
}
