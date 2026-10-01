package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/geoo115/E-commerceMicroservices/api-gateway/internal/middleware"
	cartv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/cart/v1"
	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
)

// Every cart operation uses the user ID from the JWT, never from the request,
// so a user can only ever read or modify their own cart.

// GetCart returns the caller's cart. GET /api/v1/cart
func (h *Handler) GetCart(c *gin.Context) {
	resp, err := h.clients.Cart.GetCart(c.Request.Context(), &cartv1.GetCartRequest{UserId: middleware.UserID(c)})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toCart(resp.GetCart()))
}

type addCartItemRequest struct {
	ProductID uint64 `json:"product_id" binding:"required"`
	Quantity  int32  `json:"quantity" binding:"required,min=1,max=100"`
}

// AddCartItem adds a product to the cart. POST /api/v1/cart/items
func (h *Handler) AddCartItem(c *gin.Context) {
	var req addCartItemRequest
	if !bindJSON(c, &req) {
		return
	}
	if !h.ensureProductExists(c, req.ProductID) {
		return
	}
	resp, err := h.clients.Cart.AddItem(c.Request.Context(), &cartv1.AddItemRequest{
		UserId:    middleware.UserID(c),
		ProductId: req.ProductID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toCart(resp.GetCart()))
}

type updateCartItemRequest struct {
	Quantity int32 `json:"quantity" binding:"min=0,max=100"`
}

// UpdateCartItem sets a product's quantity; 0 removes it. PUT /api/v1/cart/items/:productId
func (h *Handler) UpdateCartItem(c *gin.Context) {
	productID, ok := pathID(c, "productId")
	if !ok {
		return
	}
	var req updateCartItemRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.clients.Cart.UpdateItem(c.Request.Context(), &cartv1.UpdateItemRequest{
		UserId:    middleware.UserID(c),
		ProductId: productID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toCart(resp.GetCart()))
}

// RemoveCartItem removes a product from the cart. DELETE /api/v1/cart/items/:productId
func (h *Handler) RemoveCartItem(c *gin.Context) {
	productID, ok := pathID(c, "productId")
	if !ok {
		return
	}
	resp, err := h.clients.Cart.RemoveItem(c.Request.Context(), &cartv1.RemoveItemRequest{
		UserId:    middleware.UserID(c),
		ProductId: productID,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toCart(resp.GetCart()))
}

// ClearCart empties the cart. DELETE /api/v1/cart
func (h *Handler) ClearCart(c *gin.Context) {
	if _, err := h.clients.Cart.ClearCart(c.Request.Context(), &cartv1.ClearCartRequest{UserId: middleware.UserID(c)}); err != nil {
		h.grpcError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Checkout turns the cart into an order, then empties the cart. POST /api/v1/cart/checkout
func (h *Handler) Checkout(c *gin.Context) {
	userID := middleware.UserID(c)
	cart, err := h.clients.Cart.GetCart(c.Request.Context(), &cartv1.GetCartRequest{UserId: userID})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	if len(cart.GetCart().GetItems()) == 0 {
		badRequest(c, "cart is empty")
		return
	}

	in := &orderv1.CreateOrderRequest{UserId: userID}
	for _, it := range cart.GetCart().GetItems() {
		in.Items = append(in.Items, &orderv1.CreateOrderRequest_Item{ProductId: it.GetProductId(), Quantity: it.GetQuantity()})
	}
	order, err := h.clients.Order.CreateOrder(c.Request.Context(), in)
	if err != nil {
		h.grpcError(c, err)
		return
	}

	// The order exists at this point; failing to clear the cart must not fail the checkout.
	if _, err := h.clients.Cart.ClearCart(c.Request.Context(), &cartv1.ClearCartRequest{UserId: userID}); err != nil {
		h.log.WarnContext(c.Request.Context(), "cart not cleared after checkout", "user_id", userID, "error", err)
	}
	c.JSON(http.StatusCreated, toOrder(order.GetOrder()))
}
