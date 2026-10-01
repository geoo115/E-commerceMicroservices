package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
	cartv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/cart/v1"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	paymentv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/payment/v1"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
)

// The generated gRPC clients are interfaces, so tests embed them and override
// only the methods they need. Calling anything else panics, which flags
// unexpected backend calls.

type fakeAuth struct {
	authv1.AuthServiceClient
	err error
}

// ValidateToken accepts "customer-<id>" and "admin-<id>" tokens.
func (f fakeAuth) ValidateToken(_ context.Context, in *authv1.ValidateTokenRequest, _ ...grpc.CallOption) (*authv1.ValidateTokenResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	switch in.GetToken() {
	case "customer-1":
		return &authv1.ValidateTokenResponse{UserId: 1, Role: "customer"}, nil
	case "customer-2":
		return &authv1.ValidateTokenResponse{UserId: 2, Role: "customer"}, nil
	case "admin-9":
		return &authv1.ValidateTokenResponse{UserId: 9, Role: "admin"}, nil
	}
	return nil, status.Error(codes.Unauthenticated, "invalid token")
}

type fakeOrders struct {
	orderv1.OrderServiceClient
	order     *orderv1.Order
	err       error
	lastInput *orderv1.CreateOrderRequest
}

func (f *fakeOrders) CreateOrder(_ context.Context, in *orderv1.CreateOrderRequest, _ ...grpc.CallOption) (*orderv1.CreateOrderResponse, error) {
	f.lastInput = in
	if f.err != nil {
		return nil, f.err
	}
	return &orderv1.CreateOrderResponse{Order: &orderv1.Order{Id: 1, UserId: in.GetUserId(), Status: orderv1.OrderStatus_ORDER_STATUS_PENDING}}, nil
}

func (f *fakeOrders) GetOrder(context.Context, *orderv1.GetOrderRequest, ...grpc.CallOption) (*orderv1.GetOrderResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &orderv1.GetOrderResponse{Order: f.order}, nil
}

type fakePayments struct {
	paymentv1.PaymentServiceClient
	status paymentv1.PaymentStatus
}

func (f fakePayments) ProcessPayment(_ context.Context, in *paymentv1.ProcessPaymentRequest, _ ...grpc.CallOption) (*paymentv1.ProcessPaymentResponse, error) {
	return &paymentv1.ProcessPaymentResponse{Payment: &paymentv1.Payment{OrderId: in.GetOrderId(), Status: f.status}}, nil
}

type fakeCart struct {
	cartv1.CartServiceClient
	items   []*cartv1.CartItem
	cleared bool
}

func (f *fakeCart) GetCart(_ context.Context, in *cartv1.GetCartRequest, _ ...grpc.CallOption) (*cartv1.GetCartResponse, error) {
	return &cartv1.GetCartResponse{Cart: &cartv1.Cart{UserId: in.GetUserId(), Items: f.items}}, nil
}

func (f *fakeCart) ClearCart(context.Context, *cartv1.ClearCartRequest, ...grpc.CallOption) (*cartv1.ClearCartResponse, error) {
	f.cleared = true
	return &cartv1.ClearCartResponse{}, nil
}

type fakeProducts struct {
	productv1.ProductServiceClient
	err error
}

func (f fakeProducts) CreateProduct(context.Context, *productv1.CreateProductRequest, ...grpc.CallOption) (*productv1.CreateProductResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &productv1.CreateProductResponse{Product: &productv1.Product{Id: 1}}, nil
}

func newTestRouter(c Clients) *gin.Engine {
	gin.SetMode(gin.TestMode)
	if c.Auth == nil {
		c.Auth = fakeAuth{}
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(New(c, log), log)
}

func do(r http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) (string, string) {
	t.Helper()
	var body errorBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return body.Error.Code, body.Error.Message
}

func TestAuthentication(t *testing.T) {
	tests := []struct {
		name   string
		auth   fakeAuth
		header string
		want   int
	}{
		{"no header", fakeAuth{}, "", http.StatusUnauthorized},
		{"wrong scheme", fakeAuth{}, "Basic abc", http.StatusUnauthorized},
		{"invalid token", fakeAuth{}, "Bearer nope", http.StatusUnauthorized},
		{"auth service down", fakeAuth{err: status.Error(codes.Unavailable, "down")}, "Bearer customer-1", http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRouter(Clients{Auth: tt.auth, Cart: &fakeCart{}})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.want, w.Code)
		})
	}
}

func TestAdminRoutesRequireAdminRole(t *testing.T) {
	r := newTestRouter(Clients{Product: fakeProducts{}})
	body := `{"name":"Keyboard","price_cents":1000,"category_id":1}`

	assert.Equal(t, http.StatusForbidden, do(r, http.MethodPost, "/api/v1/products", "customer-1", body).Code)
	assert.Equal(t, http.StatusCreated, do(r, http.MethodPost, "/api/v1/products", "admin-9", body).Code)
}

func TestCreateOrder_UsesTokenIdentityAndNoClientPrices(t *testing.T) {
	orders := &fakeOrders{}
	r := newTestRouter(Clients{Order: orders})

	w := do(r, http.MethodPost, "/api/v1/orders", "customer-1",
		`{"user_id":999,"items":[{"product_id":5,"quantity":2,"price":0.01}]}`)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, uint64(1), orders.lastInput.GetUserId(), "user comes from the JWT, not the body")
	assert.Equal(t, int32(2), orders.lastInput.GetItems()[0].GetQuantity())
}

func TestGetOrder_Ownership(t *testing.T) {
	orders := &fakeOrders{order: &orderv1.Order{Id: 3, UserId: 1}}
	r := newTestRouter(Clients{Order: orders})

	assert.Equal(t, http.StatusOK, do(r, http.MethodGet, "/api/v1/orders/3", "customer-1", "").Code)
	assert.Equal(t, http.StatusNotFound, do(r, http.MethodGet, "/api/v1/orders/3", "customer-2", "").Code,
		"another user's order is reported as not found")
	assert.Equal(t, http.StatusOK, do(r, http.MethodGet, "/api/v1/orders/3", "admin-9", "").Code)
	assert.Equal(t, http.StatusBadRequest, do(r, http.MethodGet, "/api/v1/orders/abc", "customer-1", "").Code)
}

func TestGRPCErrorMapping(t *testing.T) {
	tests := []struct {
		code       codes.Code
		wantStatus int
	}{
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.NotFound, http.StatusNotFound},
		{codes.AlreadyExists, http.StatusConflict},
		{codes.FailedPrecondition, http.StatusConflict},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{codes.Internal, http.StatusInternalServerError},
		{codes.Unknown, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			r := newTestRouter(Clients{Order: &fakeOrders{err: status.Error(tt.code, "secret detail")}})
			w := do(r, http.MethodPost, "/api/v1/orders", "customer-1", `{"items":[{"product_id":1,"quantity":1}]}`)
			assert.Equal(t, tt.wantStatus, w.Code)

			_, msg := errorCode(t, w)
			if tt.wantStatus == http.StatusInternalServerError {
				assert.Equal(t, "internal error", msg, "internal details must not leak")
			} else {
				assert.Equal(t, "secret detail", msg)
			}
		})
	}
}

func TestValidationErrorsUseJSONFieldNames(t *testing.T) {
	r := newTestRouter(Clients{Product: fakeProducts{}})
	w := do(r, http.MethodPost, "/api/v1/products", "admin-9", `{"name":"Keyboard","price_cents":0,"category_id":1}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, msg := errorCode(t, w)
	assert.Contains(t, msg, "price_cents")

	w = do(r, http.MethodPost, "/api/v1/products", "admin-9", `{not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPayOrder_DeclinedReturns402(t *testing.T) {
	r := newTestRouter(Clients{Payment: fakePayments{status: paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED}})
	w := do(r, http.MethodPost, "/api/v1/orders/4/payment", "customer-1", `{"method":"card","card_last_four":"0002"}`)
	assert.Equal(t, http.StatusPaymentRequired, w.Code)

	r = newTestRouter(Clients{Payment: fakePayments{status: paymentv1.PaymentStatus_PAYMENT_STATUS_SUCCEEDED}})
	w = do(r, http.MethodPost, "/api/v1/orders/4/payment", "customer-1", `{"method":"card","card_last_four":"4242"}`)
	assert.Equal(t, http.StatusCreated, w.Code)

	w = do(r, http.MethodPost, "/api/v1/orders/4/payment", "customer-1", `{"method":"cash"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCheckout(t *testing.T) {
	empty := &fakeCart{}
	r := newTestRouter(Clients{Cart: empty, Order: &fakeOrders{}})
	assert.Equal(t, http.StatusBadRequest, do(r, http.MethodPost, "/api/v1/cart/checkout", "customer-1", "").Code)

	cart := &fakeCart{items: []*cartv1.CartItem{{ProductId: 1, Quantity: 2}}}
	orders := &fakeOrders{}
	r = newTestRouter(Clients{Cart: cart, Order: orders})
	w := do(r, http.MethodPost, "/api/v1/cart/checkout", "customer-1", "")

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, uint64(1), orders.lastInput.GetItems()[0].GetProductId())
	assert.True(t, cart.cleared, "cart is emptied after a successful checkout")
}

func TestOrderJSONUsesReadableStatus(t *testing.T) {
	r := newTestRouter(Clients{Order: &fakeOrders{order: &orderv1.Order{
		Id: 3, UserId: 1, Status: orderv1.OrderStatus_ORDER_STATUS_CONFIRMED,
	}}})
	w := do(r, http.MethodGet, "/api/v1/orders/3", "customer-1", "")
	require.Equal(t, http.StatusOK, w.Code)

	var got orderDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "confirmed", got.Status)
}

func TestAuthEndpointsAreRateLimited(t *testing.T) {
	r := newTestRouter(Clients{Auth: fakeAuth{}})
	var last int
	for range 11 {
		last = do(r, http.MethodPost, "/api/v1/auth/login", "", `{}`).Code
	}
	assert.Equal(t, http.StatusTooManyRequests, last)
}
