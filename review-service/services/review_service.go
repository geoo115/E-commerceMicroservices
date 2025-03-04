package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/geoo115/E-commerceMicroservices/message-broker/producers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/topics"
	"github.com/geoo115/E-commerceMicroservices/review-service/cache"
	"github.com/geoo115/E-commerceMicroservices/review-service/db"
	"github.com/geoo115/E-commerceMicroservices/review-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/review-service/proto"
	"gorm.io/gorm"
)

type ReviewServer struct {
	pb.UnimplementedReviewServiceServer
}

func NewReviewServer() *ReviewServer {
	return &ReviewServer{}
}

func reviewCacheKey(reviewID uint) string {
	return fmt.Sprintf("review:%d", reviewID)
}

func reviewsByProductCacheKey(productID uint64) string {
	return fmt.Sprintf("reviews:product:%d", productID)
}

func wishlistCacheKey(userID uint64) string {
	return fmt.Sprintf("wishlist:user:%d", userID)
}

func (s *ReviewServer) CreateReview(ctx context.Context, req *pb.CreateReviewRequest) (*pb.ReviewResponse, error) {
	review := models.Review{
		UserID:    uint(req.UserId),
		ProductID: uint(req.ProductId),
		Rating:    int(req.Rating),
		Comment:   req.Comment,
	}
	if err := db.DB.Create(&review).Error; err != nil {
		return nil, fmt.Errorf("failed to create review: %v", err)
	}
	eventPayload, err := json.Marshal(review)
	if err != nil {
		log.Printf("Failed to marshal review event: %v", err)
	} else {
		if err := producers.PublishEvent(topics.ReviewAdded, eventPayload); err != nil {
			log.Printf("Failed to publish review event: %v", err)
		}
	}
	return &pb.ReviewResponse{
		ReviewId:  uint64(review.ID),
		UserId:    uint64(review.UserID),
		ProductId: uint64(review.ProductID),
		Rating:    int32(review.Rating),
		Comment:   review.Comment,
	}, nil
}

// GetReview retrieves a review by its ID, using caching if available.
func (s *ReviewServer) GetReview(ctx context.Context, req *pb.GetReviewRequest) (*pb.ReviewResponse, error) {
	key := reviewCacheKey(uint(req.ReviewId))
	cached, err := cache.RedisClient.Get(ctx, key).Result()
	if err == nil && cached != "" {
		var reviewResp pb.ReviewResponse
		if err := json.Unmarshal([]byte(cached), &reviewResp); err == nil {
			return &reviewResp, nil
		}
		// If unmarshalling fails, fall back to DB.
	}

	var review models.Review
	if err := db.DB.First(&review, req.ReviewId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("review not found")
		}
		return nil, fmt.Errorf("failed to get review: %v", err)
	}

	resp := &pb.ReviewResponse{
		ReviewId:  uint64(review.ID),
		UserId:    uint64(review.UserID),
		ProductId: uint64(review.ProductID),
		Rating:    int32(review.Rating),
		Comment:   review.Comment,
	}

	// Cache the response for 5 minutes.
	if bytes, err := json.Marshal(resp); err == nil {
		cache.RedisClient.Set(ctx, key, bytes, 5*time.Minute)
	}

	return resp, nil
}

// ListReviews lists all reviews for a given product, using caching if available.
func (s *ReviewServer) ListReviews(ctx context.Context, req *pb.ListReviewsRequest) (*pb.ListReviewsResponse, error) {
	key := reviewsByProductCacheKey(req.ProductId)
	cached, err := cache.RedisClient.Get(ctx, key).Result()
	if err == nil && cached != "" {
		var listResp pb.ListReviewsResponse
		if err := json.Unmarshal([]byte(cached), &listResp); err == nil {
			return &listResp, nil
		}
		// Fall back to DB if unmarshal fails.
	}

	var reviews []models.Review
	if err := db.DB.Where("product_id = ?", req.ProductId).Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("failed to list reviews: %v", err)
	}

	res := &pb.ListReviewsResponse{}
	for _, r := range reviews {
		res.Reviews = append(res.Reviews, &pb.ReviewResponse{
			ReviewId:  uint64(r.ID),
			UserId:    uint64(r.UserID),
			ProductId: uint64(r.ProductID),
			Rating:    int32(r.Rating),
			Comment:   r.Comment,
		})
	}

	// Cache the list response for 5 minutes.
	if bytes, err := json.Marshal(res); err == nil {
		cache.RedisClient.Set(ctx, key, bytes, 5*time.Minute)
	}

	return res, nil
}

// DeleteReview deletes a review and publishes a review_deleted event.
func (s *ReviewServer) DeleteReview(ctx context.Context, req *pb.DeleteReviewRequest) (*pb.ReviewResponse, error) {
	var review models.Review
	if err := db.DB.First(&review, req.ReviewId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("review not found")
		}
		return nil, fmt.Errorf("failed to find review: %v", err)
	}

	if err := db.DB.Delete(&review).Error; err != nil {
		return nil, fmt.Errorf("failed to delete review: %v", err)
	}

	// Invalidate caches.
	cache.RedisClient.Del(ctx, reviewCacheKey(uint(req.ReviewId)))
	cache.RedisClient.Del(ctx, reviewsByProductCacheKey(uint64(review.ProductID)))

	// Publish review_deleted event.
	delEvent := map[string]interface{}{
		"action":    "review_deleted",
		"reviewId":  review.ID,
		"productId": review.ProductID,
		"userId":    review.UserID,
	}
	eventPayload, err := json.Marshal(delEvent)
	if err != nil {
		log.Printf("Failed to marshal review_deleted event: %v", err)
	} else {
		if err := producers.PublishEvent(topics.ReviewDeleted, eventPayload); err != nil {
			log.Printf("Failed to publish review_deleted event: %v", err)
		}
	}

	return &pb.ReviewResponse{
		ReviewId:  uint64(review.ID),
		UserId:    uint64(review.UserID),
		ProductId: uint64(review.ProductID),
		Rating:    int32(review.Rating),
		Comment:   review.Comment,
	}, nil
}

// AddToWishlist adds a product to a user's wishlist and publishes a wishlist_updated event.
func (s *ReviewServer) AddToWishlist(ctx context.Context, req *pb.AddToWishlistRequest) (*pb.WishlistResponse, error) {
	uniqueIndex := fmt.Sprintf("%d-%d", req.UserId, req.ProductId)
	wishlist := models.Wishlist{
		UserID:      uint(req.UserId),
		ProductID:   uint(req.ProductId),
		UniqueIndex: uniqueIndex,
	}

	if err := db.DB.Create(&wishlist).Error; err != nil {
		return nil, fmt.Errorf("failed to add to wishlist: %v", err)
	}

	// Invalidate wishlist cache for the user.
	cache.RedisClient.Del(ctx, wishlistCacheKey(req.UserId))

	// Publish wishlist_updated event.
	wlEvent := map[string]interface{}{
		"action":     "wishlist_item_added",
		"wishlistId": wishlist.ID,
		"userId":     wishlist.UserID,
		"productId":  wishlist.ProductID,
	}
	eventPayload, err := json.Marshal(wlEvent)
	if err != nil {
		log.Printf("Failed to marshal wishlist add event: %v", err)
	} else {
		if err := producers.PublishEvent(topics.WishlistUpdated, eventPayload); err != nil {
			log.Printf("Failed to publish wishlist add event: %v", err)
		}
	}

	return &pb.WishlistResponse{
		WishlistId: uint64(wishlist.ID),
		UserId:     uint64(wishlist.UserID),
		ProductId:  uint64(wishlist.ProductID),
	}, nil
}

// RemoveFromWishlist removes a product from a user's wishlist and publishes a wishlist_updated event.
func (s *ReviewServer) RemoveFromWishlist(ctx context.Context, req *pb.RemoveFromWishlistRequest) (*pb.WishlistResponse, error) {
	uniqueIndex := fmt.Sprintf("%d-%d", req.UserId, req.ProductId)
	var wishlist models.Wishlist
	if err := db.DB.Where("unique_index = ?", uniqueIndex).First(&wishlist).Error; err != nil {
		return nil, fmt.Errorf("wishlist item not found: %v", err)
	}

	if err := db.DB.Delete(&wishlist).Error; err != nil {
		return nil, fmt.Errorf("failed to remove from wishlist: %v", err)
	}

	// Invalidate wishlist cache for the user.
	cache.RedisClient.Del(ctx, wishlistCacheKey(req.UserId))

	// Publish wishlist_updated event.
	wlEvent := map[string]interface{}{
		"action":     "wishlist_item_removed",
		"wishlistId": wishlist.ID,
		"userId":     wishlist.UserID,
		"productId":  wishlist.ProductID,
	}
	eventPayload, err := json.Marshal(wlEvent)
	if err != nil {
		log.Printf("Failed to marshal wishlist remove event: %v", err)
	} else {
		if err := producers.PublishEvent(topics.WishlistUpdated, eventPayload); err != nil {
			log.Printf("Failed to publish wishlist remove event: %v", err)
		}
	}

	return &pb.WishlistResponse{
		WishlistId: uint64(wishlist.ID),
		UserId:     uint64(wishlist.UserID),
		ProductId:  uint64(wishlist.ProductID),
	}, nil
}

// GetWishlist retrieves all wishlist items for a user, using caching if available.
func (s *ReviewServer) GetWishlist(ctx context.Context, req *pb.GetWishlistRequest) (*pb.WishlistListResponse, error) {
	key := wishlistCacheKey(req.UserId)
	cached, err := cache.RedisClient.Get(ctx, key).Result()
	if err == nil && cached != "" {
		var wishlistResp pb.WishlistListResponse
		if err := json.Unmarshal([]byte(cached), &wishlistResp); err == nil {
			return &wishlistResp, nil
		}
		// Fall back to DB if unmarshal fails.
	}

	var wishlistItems []models.Wishlist
	if err := db.DB.Where("user_id = ?", req.UserId).Find(&wishlistItems).Error; err != nil {
		return nil, fmt.Errorf("failed to get wishlist: %v", err)
	}

	res := &pb.WishlistListResponse{}
	for _, item := range wishlistItems {
		res.Items = append(res.Items, &pb.WishlistResponse{
			WishlistId: uint64(item.ID),
			UserId:     uint64(item.UserID),
			ProductId:  uint64(item.ProductID),
		})
	}

	// Cache the wishlist response for 5 minutes.
	if bytes, err := json.Marshal(res); err == nil {
		cache.RedisClient.Set(ctx, key, bytes, 5*time.Minute)
	}

	return res, nil
}

func HandleReviewAddedEvent(message []byte) {
	log.Printf("Handling review_added event: %s", string(message))
	// TODO: Process the review added event as needed.
}

// HandleReviewDeletedEvent processes a review_deleted event.
func HandleReviewDeletedEvent(message []byte) {
	log.Printf("Handling review_deleted event: %s", string(message))
	// TODO: Process the review deleted event as needed.
}
