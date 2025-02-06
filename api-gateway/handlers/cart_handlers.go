package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	pbCart "github.com/geoo115/E-commerceMicroservices/cart-service/proto"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// getCartServiceClient establishes a gRPC connection to the cart service.
func getCartServiceClient() (pbCart.CartServiceClient, *grpc.ClientConn, error) {
	addr := viper.GetString("cart_service.address")
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return pbCart.NewCartServiceClient(conn), conn, nil
}

// GetCart retrieves the cart for a given user.
func GetCart(c *gin.Context) {
	userId := c.Param("userId")
	uid, err := strconv.ParseUint(userId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	client, conn, err := getCartServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to cart service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.GetCart(ctx, &pbCart.GetCartRequest{UserId: uid})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// AddToCart adds an item to the user's cart.
func AddToCart(c *gin.Context) {
	var req pbCart.AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, conn, err := getCartServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to cart service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.AddItemToCart(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RemoveFromCart removes an item from the user's cart.
func RemoveFromCart(c *gin.Context) {
	var req pbCart.RemoveItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, conn, err := getCartServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to cart service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.RemoveCartItem(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ClearCart clears all items from a user's cart.
func ClearCart(c *gin.Context) {
	userId := c.Param("userId")
	uid, err := strconv.ParseUint(userId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	client, conn, err := getCartServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to cart service"})
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.ClearCart(ctx, &pbCart.ClearCartRequest{UserId: uint32(uid)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
