package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/geoo115/E-commerceMicroservices/cart-service/cache"
	"github.com/geoo115/E-commerceMicroservices/cart-service/db"
	"github.com/geoo115/E-commerceMicroservices/cart-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/cart-service/proto"

	"gorm.io/gorm"
)

// CartServer implements the gRPC CartService.
type CartServer struct {
	pb.UnimplementedCartServiceServer
}

// NewCartServer creates and returns a new CartServer instance.
func NewCartServer() *CartServer {
	return &CartServer{}
}

// cacheKey returns the Redis cache key for a given user.
func cacheKey(userID uint64) string {
	return fmt.Sprintf("cart:%d", userID)
}

// AddItemToCart adds an item to the cart.
func (s *CartServer) AddItemToCart(ctx context.Context, req *pb.AddItemRequest) (*pb.CartResponse, error) {
	var cartItem models.Cart

	// Check if the item already exists in the cart.
	result := db.DB.Where("user_id = ? AND product_id = ?", req.UserId, req.ProductId).First(&cartItem)
	if result.Error == nil {
		// Item exists; update the quantity.
		cartItem.Quantity += int(req.Quantity)
		if err := db.DB.Save(&cartItem).Error; err != nil {
			return nil, fmt.Errorf("failed to update cart item: %v", err)
		}
	} else if result.Error == gorm.ErrRecordNotFound {
		// Item does not exist; create a new entry.
		cartItem = models.Cart{
			UserID:    uint(req.UserId),
			ProductID: uint(req.ProductId),
			Quantity:  int(req.Quantity),
		}
		if err := db.DB.Create(&cartItem).Error; err != nil {
			return nil, fmt.Errorf("failed to add item to cart: %v", err)
		}
	} else {
		return nil, fmt.Errorf("database error: %v", result.Error)
	}

	// Flush cached cart for this user.
	cache.RedisClient.Del(ctx, cacheKey(req.UserId))

	return &pb.CartResponse{
		UserId:    uint64(cartItem.UserID),
		ProductId: uint64(cartItem.ProductID),
		Quantity:  int32(cartItem.Quantity),
	}, nil
}

// UpdateCartItem updates the quantity of an existing cart item.
func (s *CartServer) UpdateCartItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.CartResponse, error) {
	var cartItem models.Cart

	if err := db.DB.Where("user_id = ? AND product_id = ?", req.UserId, req.ProductId).First(&cartItem).Error; err != nil {
		return nil, fmt.Errorf("cart item not found")
	}

	// Update quantity.
	cartItem.Quantity = int(req.Quantity)
	if err := db.DB.Save(&cartItem).Error; err != nil {
		return nil, fmt.Errorf("failed to update cart item: %v", err)
	}

	// Flush cached cart for this user.
	cache.RedisClient.Del(ctx, cacheKey(req.UserId))

	return &pb.CartResponse{
		UserId:    uint64(cartItem.UserID),
		ProductId: uint64(cartItem.ProductID),
		Quantity:  int32(cartItem.Quantity),
	}, nil
}

// RemoveCartItem removes an item from the cart.
func (s *CartServer) RemoveCartItem(ctx context.Context, req *pb.RemoveItemRequest) (*pb.CartResponse, error) {
	if err := db.DB.Where("user_id = ? AND product_id = ?", req.UserId, req.ProductId).Delete(&models.Cart{}).Error; err != nil {
		return nil, fmt.Errorf("failed to remove item from cart: %v", err)
	}

	// Flush cached cart for this user.
	cache.RedisClient.Del(ctx, cacheKey(req.UserId))

	return &pb.CartResponse{
		UserId:    req.UserId,
		ProductId: req.ProductId,
		Quantity:  0,
	}, nil
}

// GetCart retrieves all cart items for a user.
func (s *CartServer) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.CartListResponse, error) {
	cacheKey := cacheKey(req.UserId)
	// Attempt to fetch the cart from Redis.
	cached, err := cache.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var response pb.CartListResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			return &response, nil
		}
		// If unmarshal fails, fall back to DB.
	}

	// Fetch from the database.
	var cartItems []models.Cart
	if err := db.DB.Where("user_id = ?", req.UserId).Find(&cartItems).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch cart items: %v", err)
	}

	response := &pb.CartListResponse{}
	for _, item := range cartItems {
		response.Items = append(response.Items, &pb.CartResponse{
			UserId:    uint64(item.UserID),
			ProductId: uint64(item.ProductID),
			Quantity:  int32(item.Quantity),
		})
	}

	// Cache the response for 5 minutes.
	if cartBytes, err := json.Marshal(response); err == nil {
		cache.RedisClient.Set(ctx, cacheKey, cartBytes, 5*time.Minute)
	}

	return response, nil
}

// ClearCart clears all items from a user's cart.
func (s *CartServer) ClearCart(ctx context.Context, req *pb.ClearCartRequest) (*pb.CartClearResponse, error) {
	if err := db.DB.Where("user_id = ?", req.UserId).Delete(&models.Cart{}).Error; err != nil {
		return nil, fmt.Errorf("failed to clear cart: %v", err)
	}

	// Flush cached cart for this user.
	cache.RedisClient.Del(ctx, cacheKey(req.UserId))

	return &pb.CartClearResponse{
		Message: "Cart cleared successfully",
	}, nil
}
