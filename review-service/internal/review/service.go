// Package review implements review-service: product reviews and wishlists.
package review

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/paging"
	reviewv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/review/v1"
)

const maxCommentLength = 2000

// Review is a user's rating of a product. A user can review a product once.
type Review struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"not null;uniqueIndex:idx_review_user_product"`
	ProductID uint64 `gorm:"not null;uniqueIndex:idx_review_user_product;index"`
	Rating    int32  `gorm:"not null;check:rating BETWEEN 1 AND 5"`
	Comment   string `gorm:"type:text"`
	CreatedAt time.Time
}

// WishlistItem is a product saved by a user.
type WishlistItem struct {
	UserID    uint64 `gorm:"primaryKey;autoIncrement:false"`
	ProductID uint64 `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt time.Time
}

// Models lists the tables owned by review-service.
var Models = []any{&Review{}, &WishlistItem{}, &events.OutboxMessage{}}

// Service implements reviewv1.ReviewServiceServer.
type Service struct {
	reviewv1.UnimplementedReviewServiceServer

	db  *gorm.DB
	log *slog.Logger
}

// NewService returns a review Service.
func NewService(db *gorm.DB, log *slog.Logger) *Service {
	return &Service{db: db, log: log}
}

// CreateReview stores a review and emits review.created (product-service
// uses it to maintain the product's average rating).
func (s *Service) CreateReview(ctx context.Context, req *reviewv1.CreateReviewRequest) (*reviewv1.CreateReviewResponse, error) {
	if req.GetUserId() == 0 || req.GetProductId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and product_id are required")
	}
	if req.GetRating() < 1 || req.GetRating() > 5 {
		return nil, status.Error(codes.InvalidArgument, "rating must be between 1 and 5")
	}
	comment := strings.TrimSpace(req.GetComment())
	if len(comment) > maxCommentLength {
		return nil, status.Errorf(codes.InvalidArgument, "comment must be at most %d characters", maxCommentLength)
	}

	r := Review{UserID: req.GetUserId(), ProductID: req.GetProductId(), Rating: req.GetRating(), Comment: comment}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&r).Error; err != nil {
			return err
		}
		return enqueue(tx, events.ReviewCreated, r)
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil, status.Error(codes.AlreadyExists, "you have already reviewed this product")
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "create review", err)
	}
	return &reviewv1.CreateReviewResponse{Review: reviewToProto(r)}, nil
}

// ListReviews returns a page of reviews for a product, newest first.
func (s *Service) ListReviews(ctx context.Context, req *reviewv1.ListReviewsRequest) (*reviewv1.ListReviewsResponse, error) {
	q := s.db.WithContext(ctx).Model(&Review{}).Where("product_id = ?", req.GetProductId())
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "count reviews", err)
	}
	limit, offset := paging.Normalize(req.GetPage(), req.GetPageSize())
	var reviews []Review
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&reviews).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "list reviews", err)
	}
	out := make([]*reviewv1.Review, len(reviews))
	for i, r := range reviews {
		out[i] = reviewToProto(r)
	}
	return &reviewv1.ListReviewsResponse{Reviews: out, Total: total}, nil
}

// DeleteReview deletes a review if the requester is its author or an admin.
func (s *Service) DeleteReview(ctx context.Context, req *reviewv1.DeleteReviewRequest) (*reviewv1.DeleteReviewResponse, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var r Review
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&r, req.GetReviewId()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "review not found")
			}
			return err
		}
		if r.UserID != req.GetRequesterId() && !req.GetRequesterIsAdmin() {
			return status.Error(codes.PermissionDenied, "only the author or an admin can delete this review")
		}
		if err := tx.Delete(&r).Error; err != nil {
			return err
		}
		return enqueue(tx, events.ReviewDeleted, r)
	})
	if _, ok := status.FromError(err); ok && err != nil {
		return nil, err
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "delete review", err)
	}
	return &reviewv1.DeleteReviewResponse{}, nil
}

// AddToWishlist saves a product for the user. Adding it twice is a no-op.
func (s *Service) AddToWishlist(ctx context.Context, req *reviewv1.AddToWishlistRequest) (*reviewv1.AddToWishlistResponse, error) {
	if req.GetUserId() == 0 || req.GetProductId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and product_id are required")
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&WishlistItem{UserID: req.GetUserId(), ProductID: req.GetProductId()}).Error
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "add to wishlist", err)
	}
	return &reviewv1.AddToWishlistResponse{}, nil
}

// RemoveFromWishlist removes a product from the user's wishlist.
func (s *Service) RemoveFromWishlist(ctx context.Context, req *reviewv1.RemoveFromWishlistRequest) (*reviewv1.RemoveFromWishlistResponse, error) {
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND product_id = ?", req.GetUserId(), req.GetProductId()).
		Delete(&WishlistItem{}).Error
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "remove from wishlist", err)
	}
	return &reviewv1.RemoveFromWishlistResponse{}, nil
}

// ListWishlist returns the user's wishlist, most recent first.
func (s *Service) ListWishlist(ctx context.Context, req *reviewv1.ListWishlistRequest) (*reviewv1.ListWishlistResponse, error) {
	var items []WishlistItem
	if err := s.db.WithContext(ctx).Where("user_id = ?", req.GetUserId()).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "list wishlist", err)
	}
	out := make([]*reviewv1.WishlistItem, len(items))
	for i, it := range items {
		out[i] = &reviewv1.WishlistItem{ProductId: it.ProductID, AddedAt: timestamppb.New(it.CreatedAt)}
	}
	return &reviewv1.ListWishlistResponse{Items: out}, nil
}

func enqueue(tx *gorm.DB, eventType string, r Review) error {
	ev, err := events.New(eventType, events.ReviewData{ReviewID: r.ID, ProductID: r.ProductID, Rating: r.Rating})
	if err != nil {
		return err
	}
	return events.Enqueue(tx, events.TopicReviews, strconv.FormatUint(r.ProductID, 10), ev)
}

func reviewToProto(r Review) *reviewv1.Review {
	return &reviewv1.Review{
		Id:        r.ID,
		UserId:    r.UserID,
		ProductId: r.ProductID,
		Rating:    r.Rating,
		Comment:   r.Comment,
		CreatedAt: timestamppb.New(r.CreatedAt),
	}
}
