package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/geoo115/E-commerceMicroservices/api-gateway/internal/middleware"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
	paymentv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/payment/v1"
)

type createOrderRequest struct {
	Items []struct {
		ProductID uint64 `json:"product_id" binding:"required"`
		Quantity  int32  `json:"quantity" binding:"required,min=1,max=100"`
	} `json:"items" binding:"required,min=1,max=50,dive"`
}

// CreateOrder places an order for the caller. POST /api/v1/orders
//
// The request carries product IDs and quantities only. Any price sent by the
// client is ignored because order-service prices items from the catalog.
func (h *Handler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if !bindJSON(c, &req) {
		return
	}
	in := &orderv1.CreateOrderRequest{UserId: middleware.UserID(c)}
	for _, it := range req.Items {
		in.Items = append(in.Items, &orderv1.CreateOrderRequest_Item{ProductId: it.ProductID, Quantity: it.Quantity})
	}
	resp, err := h.clients.Order.CreateOrder(c.Request.Context(), in)
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toOrder(resp.GetOrder()))
}

// ListMyOrders returns the caller's orders. GET /api/v1/orders
func (h *Handler) ListMyOrders(c *gin.Context) {
	pageNum, size := pageParams(c)
	resp, err := h.clients.Order.ListOrders(c.Request.Context(), &orderv1.ListOrdersRequest{
		UserId:   middleware.UserID(c),
		Page:     pageNum,
		PageSize: size,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, page[orderDTO]{
		Items:    mapSlice(resp.GetOrders(), toOrder),
		Total:    resp.GetTotal(),
		Page:     pageNum,
		PageSize: size,
	})
}

// GetOrder returns one of the caller's orders (admins can read any). GET /api/v1/orders/:id
func (h *Handler) GetOrder(c *gin.Context) {
	order, ok := h.loadOwnedOrder(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toOrder(order))
}

type cancelOrderRequest struct {
	Reason string `json:"reason" binding:"max=200"`
}

// CancelOrder cancels a pending or confirmed order. POST /api/v1/orders/:id/cancel
func (h *Handler) CancelOrder(c *gin.Context) {
	var req cancelOrderRequest
	if c.Request.ContentLength > 0 && !bindJSON(c, &req) {
		return
	}
	order, ok := h.loadOwnedOrder(c)
	if !ok {
		return
	}
	resp, err := h.clients.Order.CancelOrder(c.Request.Context(), &orderv1.CancelOrderRequest{OrderId: order.GetId(), Reason: req.Reason})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toOrder(resp.GetOrder()))
}

type updateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=shipped delivered"`
}

var fulfilmentStatuses = map[string]orderv1.OrderStatus{
	"shipped":   orderv1.OrderStatus_ORDER_STATUS_SHIPPED,
	"delivered": orderv1.OrderStatus_ORDER_STATUS_DELIVERED,
}

// UpdateOrderStatus records fulfilment progress (admin). PATCH /api/v1/orders/:id/status
func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req updateOrderStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.clients.Order.UpdateOrderStatus(c.Request.Context(), &orderv1.UpdateOrderStatusRequest{
		OrderId: id,
		Status:  fulfilmentStatuses[req.Status],
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toOrder(resp.GetOrder()))
}

type payOrderRequest struct {
	Method       string `json:"method" binding:"required,oneof=card paypal wallet"`
	CardLastFour string `json:"card_last_four" binding:"omitempty,len=4,numeric"` // required for cards; checked by payment-service
}

// PayOrder pays a confirmed order. POST /api/v1/orders/:id/payment
//
// No amount is accepted: payment-service charges the order total. A declined
// payment returns 402 with the failed payment so the client can retry.
func (h *Handler) PayOrder(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req payOrderRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.clients.Payment.ProcessPayment(c.Request.Context(), &paymentv1.ProcessPaymentRequest{
		OrderId:      id,
		UserId:       middleware.UserID(c),
		Method:       req.Method,
		CardLastFour: req.CardLastFour,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	httpStatus := http.StatusCreated
	if resp.GetPayment().GetStatus() == paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED {
		httpStatus = http.StatusPaymentRequired
	}
	c.JSON(httpStatus, toPayment(resp.GetPayment()))
}

// GetOrderPayment returns the latest payment of an order. GET /api/v1/orders/:id/payment
func (h *Handler) GetOrderPayment(c *gin.Context) {
	order, ok := h.loadOwnedOrder(c)
	if !ok {
		return
	}
	resp, err := h.clients.Payment.GetPaymentByOrder(c.Request.Context(), &paymentv1.GetPaymentByOrderRequest{OrderId: order.GetId()})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPayment(resp.GetPayment()))
}

// loadOwnedOrder loads the order in the :id path parameter and checks that the
// caller owns it (or is an admin). Orders of other users are reported as not
// found so that order IDs cannot be probed.
func (h *Handler) loadOwnedOrder(c *gin.Context) (*orderv1.Order, bool) {
	id, ok := pathID(c, "id")
	if !ok {
		return nil, false
	}
	resp, err := h.clients.Order.GetOrder(c.Request.Context(), &orderv1.GetOrderRequest{OrderId: id})
	if err != nil {
		h.grpcError(c, err)
		return nil, false
	}
	if resp.GetOrder().GetUserId() != middleware.UserID(c) && !middleware.IsAdmin(c) {
		notFound(c, "order not found")
		return nil, false
	}
	return resp.GetOrder(), true
}
