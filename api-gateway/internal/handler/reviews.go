package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/geoo115/E-commerceMicroservices/api-gateway/internal/middleware"
	reviewv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/review/v1"
)

// ListReviews returns a page of reviews for a product. GET /api/v1/products/:id/reviews
func (h *Handler) ListReviews(c *gin.Context) {
	productID, ok := pathID(c, "id")
	if !ok {
		return
	}
	pageNum, size := pageParams(c)
	resp, err := h.clients.Review.ListReviews(c.Request.Context(), &reviewv1.ListReviewsRequest{
		ProductId: productID,
		Page:      pageNum,
		PageSize:  size,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, page[reviewDTO]{
		Items:    mapSlice(resp.GetReviews(), toReview),
		Total:    resp.GetTotal(),
		Page:     pageNum,
		PageSize: size,
	})
}

type createReviewRequest struct {
	Rating  int32  `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment" binding:"max=2000"`
}

// CreateReview reviews a product as the caller. POST /api/v1/products/:id/reviews
func (h *Handler) CreateReview(c *gin.Context) {
	productID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req createReviewRequest
	if !bindJSON(c, &req) {
		return
	}
	if !h.ensureProductExists(c, productID) {
		return
	}
	resp, err := h.clients.Review.CreateReview(c.Request.Context(), &reviewv1.CreateReviewRequest{
		UserId:    middleware.UserID(c),
		ProductId: productID,
		Rating:    req.Rating,
		Comment:   req.Comment,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toReview(resp.GetReview()))
}

// DeleteReview deletes a review; review-service checks author-or-admin. DELETE /api/v1/reviews/:id
func (h *Handler) DeleteReview(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	_, err := h.clients.Review.DeleteReview(c.Request.Context(), &reviewv1.DeleteReviewRequest{
		ReviewId:         id,
		RequesterId:      middleware.UserID(c),
		RequesterIsAdmin: middleware.IsAdmin(c),
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListWishlist returns the caller's wishlist. GET /api/v1/wishlist
func (h *Handler) ListWishlist(c *gin.Context) {
	resp, err := h.clients.Review.ListWishlist(c.Request.Context(), &reviewv1.ListWishlistRequest{UserId: middleware.UserID(c)})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": mapSlice(resp.GetItems(), func(it *reviewv1.WishlistItem) wishlistItemDTO {
		return wishlistItemDTO{ProductID: it.GetProductId(), AddedAt: asTime(it.GetAddedAt())}
	})})
}

// AddToWishlist saves a product. PUT /api/v1/wishlist/:productId (idempotent)
func (h *Handler) AddToWishlist(c *gin.Context) {
	productID, ok := pathID(c, "productId")
	if !ok || !h.ensureProductExists(c, productID) {
		return
	}
	_, err := h.clients.Review.AddToWishlist(c.Request.Context(), &reviewv1.AddToWishlistRequest{
		UserId:    middleware.UserID(c),
		ProductId: productID,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// RemoveFromWishlist removes a saved product. DELETE /api/v1/wishlist/:productId
func (h *Handler) RemoveFromWishlist(c *gin.Context) {
	productID, ok := pathID(c, "productId")
	if !ok {
		return
	}
	_, err := h.clients.Review.RemoveFromWishlist(c.Request.Context(), &reviewv1.RemoveFromWishlistRequest{
		UserId:    middleware.UserID(c),
		ProductId: productID,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
