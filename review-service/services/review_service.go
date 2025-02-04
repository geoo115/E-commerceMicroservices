package services

import (
	"context"
	"fmt"

	"review-service/db"
	"review-service/models"
	pb "review-service/proto"

	"gorm.io/gorm"
)

type ReviewServer struct {
	pb.UnimplementedReviewServiceServer
}

// NewReviewServer creates and returns a new ReviewServer instance.
func NewReviewServer() *ReviewServer {
	return &ReviewServer{}
}

// CreateReview creates a new review.
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

	return &pb.ReviewResponse{
		ReviewId:  uint64(review.ID),
		UserId:    uint64(review.UserID),
		ProductId: uint64(review.ProductID),
		Rating:    int32(review.Rating),
		Comment:   review.Comment,
	}, nil
}

// GetReview retrieves a review by its ID.
func (s *ReviewServer) GetReview(ctx context.Context, req *pb.GetReviewRequest) (*pb.ReviewResponse, error) {
	var review models.Review
	if err := db.DB.First(&review, req.ReviewId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("review not found")
		}
		return nil, fmt.Errorf("failed to get review: %v", err)
	}

	return &pb.ReviewResponse{
		ReviewId:  uint64(review.ID),
		UserId:    uint64(review.UserID),
		ProductId: uint64(review.ProductID),
		Rating:    int32(review.Rating),
		Comment:   review.Comment,
	}, nil
}

// ListReviews lists all reviews for a given product.
func (s *ReviewServer) ListReviews(ctx context.Context, req *pb.ListReviewsRequest) (*pb.ListReviewsResponse, error) {
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

	return res, nil
}

// DeleteReview deletes a review.
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

	return &pb.ReviewResponse{
		ReviewId:  uint64(review.ID),
		UserId:    uint64(review.UserID),
		ProductId: uint64(review.ProductID),
		Rating:    int32(review.Rating),
		Comment:   review.Comment,
	}, nil
}

// AddToWishlist adds a product to a user's wishlist.
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

	return &pb.WishlistResponse{
		WishlistId: uint64(wishlist.ID),
		UserId:     uint64(wishlist.UserID),
		ProductId:  uint64(wishlist.ProductID),
	}, nil
}

// RemoveFromWishlist removes a product from a user's wishlist.
func (s *ReviewServer) RemoveFromWishlist(ctx context.Context, req *pb.RemoveFromWishlistRequest) (*pb.WishlistResponse, error) {
	uniqueIndex := fmt.Sprintf("%d-%d", req.UserId, req.ProductId)
	var wishlist models.Wishlist
	if err := db.DB.Where("unique_index = ?", uniqueIndex).First(&wishlist).Error; err != nil {
		return nil, fmt.Errorf("wishlist item not found: %v", err)
	}

	if err := db.DB.Delete(&wishlist).Error; err != nil {
		return nil, fmt.Errorf("failed to remove from wishlist: %v", err)
	}

	return &pb.WishlistResponse{
		WishlistId: uint64(wishlist.ID),
		UserId:     uint64(wishlist.UserID),
		ProductId:  uint64(wishlist.ProductID),
	}, nil
}

// GetWishlist retrieves all wishlist items for a user.
func (s *ReviewServer) GetWishlist(ctx context.Context, req *pb.GetWishlistRequest) (*pb.WishlistListResponse, error) {
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
	return res, nil
}
