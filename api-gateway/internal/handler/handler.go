// Package handler implements the public REST API of the gateway. Handlers
// translate HTTP/JSON into gRPC calls, enforce ownership rules and map gRPC
// status codes back to HTTP.
package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	authv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/auth/v1"
	cartv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/cart/v1"
	inventoryv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/inventory/v1"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	paymentv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/payment/v1"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
	reviewv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/review/v1"
)

// Clients groups the gRPC clients of the backend services.
type Clients struct {
	Auth      authv1.AuthServiceClient
	Product   productv1.ProductServiceClient
	Inventory inventoryv1.InventoryServiceClient
	Cart      cartv1.CartServiceClient
	Order     orderv1.OrderServiceClient
	Payment   paymentv1.PaymentServiceClient
	Review    reviewv1.ReviewServiceClient
}

// Handler holds the dependencies of the HTTP handlers.
type Handler struct {
	clients Clients
	log     *slog.Logger
}

// New returns a Handler.
func New(clients Clients, log *slog.Logger) *Handler {
	return &Handler{clients: clients, log: log}
}

// pathID parses a positive integer path parameter.
func pathID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		badRequest(c, name+" must be a positive integer")
		return 0, false
	}
	return id, true
}

// pageParams reads ?page=&page_size= (normalized by the services).
func pageParams(c *gin.Context) (int32, int32) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
	size, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 32)
	return int32(page), int32(size)
}
