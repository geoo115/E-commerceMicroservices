package review

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/geoo115/E-commerceMicroservices/pkg/database/dbtest"
	reviewv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/review/v1"
)

func TestCreateReview_Validation(t *testing.T) {
	s := NewService(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for name, req := range map[string]*reviewv1.CreateReviewRequest{
		"rating too low":  {UserId: 1, ProductId: 1, Rating: 0},
		"rating too high": {UserId: 1, ProductId: 1, Rating: 6},
		"missing user":    {ProductId: 1, Rating: 5},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.CreateReview(context.Background(), req)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

// --- integration tests (require TEST_DATABASE_URL) ---

func TestReviewLifecycle(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t, Models...), slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := s.CreateReview(ctx, &reviewv1.CreateReviewRequest{UserId: 1, ProductId: 5, Rating: 4, Comment: "good"})
	require.NoError(t, err)
	id := created.GetReview().GetId()

	_, err = s.CreateReview(ctx, &reviewv1.CreateReviewRequest{UserId: 1, ProductId: 5, Rating: 5})
	assert.Equal(t, codes.AlreadyExists, status.Code(err), "one review per user and product")

	_, err = s.DeleteReview(ctx, &reviewv1.DeleteReviewRequest{ReviewId: id, RequesterId: 2})
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "other users cannot delete it")

	_, err = s.DeleteReview(ctx, &reviewv1.DeleteReviewRequest{ReviewId: id, RequesterId: 2, RequesterIsAdmin: true})
	require.NoError(t, err, "admins can moderate")

	_, err = s.DeleteReview(ctx, &reviewv1.DeleteReviewRequest{ReviewId: id, RequesterId: 1})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestWishlist(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t, Models...), slog.New(slog.NewTextHandler(io.Discard, nil)))

	for range 2 { // adding twice is idempotent
		_, err := s.AddToWishlist(ctx, &reviewv1.AddToWishlistRequest{UserId: 1, ProductId: 3})
		require.NoError(t, err)
	}
	list, err := s.ListWishlist(ctx, &reviewv1.ListWishlistRequest{UserId: 1})
	require.NoError(t, err)
	assert.Len(t, list.GetItems(), 1)

	_, err = s.RemoveFromWishlist(ctx, &reviewv1.RemoveFromWishlistRequest{UserId: 1, ProductId: 3})
	require.NoError(t, err)
	list, err = s.ListWishlist(ctx, &reviewv1.ListWishlistRequest{UserId: 1})
	require.NoError(t, err)
	assert.Empty(t, list.GetItems())
}
