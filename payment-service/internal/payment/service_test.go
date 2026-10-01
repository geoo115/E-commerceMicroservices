package payment

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/geoo115/E-commerceMicroservices/pkg/database/dbtest"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	paymentv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/payment/v1"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

type fakeOrders struct {
	order *orderv1.Order
	err   error
}

func (f fakeOrders) GetOrder(context.Context, *orderv1.GetOrderRequest, ...grpc.CallOption) (*orderv1.GetOrderResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &orderv1.GetOrderResponse{Order: f.order}, nil
}

func confirmedOrder(userID uint64, total int64) *orderv1.Order {
	return &orderv1.Order{Id: 10, UserId: userID, TotalCents: total, Status: orderv1.OrderStatus_ORDER_STATUS_CONFIRMED}
}

func cardRequest(userID uint64, last4 string) *paymentv1.ProcessPaymentRequest {
	return &paymentv1.ProcessPaymentRequest{OrderId: 10, UserId: userID, Method: "card", CardLastFour: last4}
}

// These cases are rejected before any database access.
func TestProcessPayment_Rejections(t *testing.T) {
	tests := []struct {
		name   string
		orders fakeOrders
		req    *paymentv1.ProcessPaymentRequest
		want   codes.Code
	}{
		{"unknown method", fakeOrders{order: confirmedOrder(1, 100)}, &paymentv1.ProcessPaymentRequest{OrderId: 10, UserId: 1, Method: "cash"}, codes.InvalidArgument},
		{"card without last four", fakeOrders{order: confirmedOrder(1, 100)}, cardRequest(1, ""), codes.InvalidArgument},
		{"someone else's order", fakeOrders{order: confirmedOrder(2, 100)}, cardRequest(1, "4242"), codes.PermissionDenied},
		{"order not found", fakeOrders{err: status.Error(codes.NotFound, "x")}, cardRequest(1, "4242"), codes.NotFound},
		{"order service down", fakeOrders{err: status.Error(codes.Unavailable, "x")}, cardRequest(1, "4242"), codes.Unavailable},
		{"order still pending", fakeOrders{order: &orderv1.Order{UserId: 1, Status: orderv1.OrderStatus_ORDER_STATUS_PENDING}}, cardRequest(1, "4242"), codes.FailedPrecondition},
		{"order cancelled", fakeOrders{order: &orderv1.Order{UserId: 1, Status: orderv1.OrderStatus_ORDER_STATUS_CANCELLED}}, cardRequest(1, "4242"), codes.FailedPrecondition},
		{"order already paid", fakeOrders{order: &orderv1.Order{UserId: 1, Status: orderv1.OrderStatus_ORDER_STATUS_PAID}}, cardRequest(1, "4242"), codes.AlreadyExists},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(nil, tt.orders, SimulatedProvider{}, "USD", discard)
			_, err := svc.ProcessPayment(context.Background(), tt.req)
			assert.Equal(t, tt.want, status.Code(err))
		})
	}
}

// --- integration tests (require TEST_DATABASE_URL) ---

func newService(t *testing.T, orders Orders) *Service {
	db := dbtest.New(t)
	require.NoError(t, Migrate(db))
	return NewService(db, orders, SimulatedProvider{}, "USD", discard)
}

func TestProcessPayment_ChargesOrderTotal(t *testing.T) {
	svc := newService(t, fakeOrders{order: confirmedOrder(1, 4999)})

	resp, err := svc.ProcessPayment(context.Background(), cardRequest(1, "4242"))
	require.NoError(t, err)
	p := resp.GetPayment()
	assert.Equal(t, paymentv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED, p.GetStatus())
	assert.Equal(t, int64(4999), p.GetAmountCents(), "amount comes from the order")
	assert.NotEmpty(t, p.GetTransactionId())

	var outbox []events.OutboxMessage
	require.NoError(t, svc.db.Find(&outbox).Error)
	assert.Len(t, outbox, 1)
}

func TestProcessPayment_DeclinedCardCanBeRetried(t *testing.T) {
	svc := newService(t, fakeOrders{order: confirmedOrder(1, 100)})

	resp, err := svc.ProcessPayment(context.Background(), cardRequest(1, DeclinedCardLastFour))
	require.NoError(t, err)
	assert.Equal(t, paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED, resp.GetPayment().GetStatus())

	resp, err = svc.ProcessPayment(context.Background(), cardRequest(1, "4242"))
	require.NoError(t, err)
	assert.Equal(t, paymentv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED, resp.GetPayment().GetStatus())
}

// TestProcessPayment_ConcurrentAttemptsChargeOnce sends 10 payments for the
// same order at once: the partial unique index lets exactly one through.
func TestProcessPayment_ConcurrentAttemptsChargeOnce(t *testing.T) {
	svc := newService(t, fakeOrders{order: confirmedOrder(1, 100)})

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		rejected  int
	)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.ProcessPayment(context.Background(), cardRequest(1, "4242"))
			mu.Lock()
			defer mu.Unlock()
			switch status.Code(err) {
			case codes.OK:
				succeeded++
			case codes.AlreadyExists:
				rejected++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, succeeded)
	assert.Equal(t, 9, rejected)
}
