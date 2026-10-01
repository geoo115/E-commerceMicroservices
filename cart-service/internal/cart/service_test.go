package cart

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/geoo115/E-commerceMicroservices/pkg/cache"
	"github.com/geoo115/E-commerceMicroservices/pkg/database/dbtest"
	cartv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/cart/v1"
)

func TestValidate(t *testing.T) {
	assert.NoError(t, validate(1, 1, 1, 1))
	assert.Equal(t, codes.InvalidArgument, status.Code(validate(0, 1, 1, 1)))
	assert.Equal(t, codes.InvalidArgument, status.Code(validate(1, 1, 0, 1)))
	assert.Equal(t, codes.InvalidArgument, status.Code(validate(1, 1, 101, 1)))
}

// --- integration tests (require TEST_DATABASE_URL) ---

func TestCartOperations(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t, Models...), cache.Noop{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := s.AddItem(ctx, &cartv1.AddItemRequest{UserId: 1, ProductId: 10, Quantity: 2})
	require.NoError(t, err)
	resp, err := s.AddItem(ctx, &cartv1.AddItemRequest{UserId: 1, ProductId: 10, Quantity: 3})
	require.NoError(t, err)
	assert.Equal(t, int32(5), resp.GetCart().GetItems()[0].GetQuantity(), "adding the same product merges lines")

	resp, err = s.AddItem(ctx, &cartv1.AddItemRequest{UserId: 1, ProductId: 10, Quantity: 100})
	require.NoError(t, err)
	assert.Equal(t, int32(maxQuantity), resp.GetCart().GetItems()[0].GetQuantity(), "quantity is capped")

	_, err = s.UpdateItem(ctx, &cartv1.UpdateItemRequest{UserId: 2, ProductId: 10, Quantity: 1})
	assert.Equal(t, codes.NotFound, status.Code(err), "users only see their own cart")

	upd, err := s.UpdateItem(ctx, &cartv1.UpdateItemRequest{UserId: 1, ProductId: 10, Quantity: 0})
	require.NoError(t, err)
	assert.Empty(t, upd.GetCart().GetItems(), "quantity 0 removes the line")
}
