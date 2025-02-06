package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	pbReview "github.com/geoo115/E-commerceMicroservices/review-service/proto"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// getReviewServiceClient establishes a gRPC connection to the review service.
func getReviewServiceClient() (pbReview.ReviewServiceClient, *grpc.ClientConn, error) {
	addr := viper.GetString("review_service.address")
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return pbReview.NewReviewServiceClient(conn), conn, nil
}

// CreateReview submits a new review by calling the CreateReview RPC.
func CreateReview(c *gin.Context) {
	var req pbReview.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, conn, err := getReviewServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to review service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.CreateReview(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListProductReviews retrieves reviews for a given product by calling the ListReviews RPC.
func ListProductReviews(c *gin.Context) {
	productId := c.Param("productId")
	pid, err := strconv.ParseUint(productId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}
	client, conn, err := getReviewServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to review service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.ListReviews(ctx, &pbReview.ListReviewsRequest{ProductId: uint64(pid)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetReview retrieves a specific review by its ID.
func GetReview(c *gin.Context) {
	reviewId := c.Param("reviewId")
	rid, err := strconv.ParseUint(reviewId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}
	client, conn, err := getReviewServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to review service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.GetReview(ctx, &pbReview.GetReviewRequest{ReviewId: uint64(rid)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// DeleteReview deletes a review by its ID.
func DeleteReview(c *gin.Context) {
	reviewId := c.Param("reviewId")
	rid, err := strconv.ParseUint(reviewId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}
	client, conn, err := getReviewServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to review service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.DeleteReview(ctx, &pbReview.DeleteReviewRequest{ReviewId: uint64(rid)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Wishlist Handlers

// AddToWishlist adds a product to a user's wishlist.
func AddToWishlist(c *gin.Context) {
	var req pbReview.AddToWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, conn, err := getReviewServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to review service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.AddToWishlist(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RemoveFromWishlist removes a product from a user's wishlist.
func RemoveFromWishlist(c *gin.Context) {
	var req pbReview.RemoveFromWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, conn, err := getReviewServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to review service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.RemoveFromWishlist(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetWishlist retrieves all wishlist items for a user.
func GetWishlist(c *gin.Context) {
	userId := c.Param("userId")
	uid, err := strconv.ParseUint(userId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	client, conn, err := getReviewServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to review service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.GetWishlist(ctx, &pbReview.GetWishlistRequest{UserId: uint64(uid)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
