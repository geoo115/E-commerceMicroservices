package product

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
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
)

func TestEscapeLike(t *testing.T) {
	assert.Equal(t, `100\% \_off\\`, escapeLike(`100% _off\`))
}

func TestRatingAverage(t *testing.T) {
	assert.Equal(t, 0.0, Product{}.RatingAverage())
	assert.Equal(t, 4.5, Product{RatingCount: 2, RatingSum: 9}.RatingAverage())
}

// --- integration tests (require TEST_DATABASE_URL) ---

func newService(t *testing.T) *Service {
	return NewService(dbtest.New(t, Models...), cache.Noop{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func createProduct(t *testing.T, s *Service, name string, price int64) *productv1.Product {
	ctx := context.Background()
	cat, err := s.CreateCategory(ctx, &productv1.CreateCategoryRequest{Name: "cat-" + name})
	require.NoError(t, err)
	resp, err := s.CreateProduct(ctx, &productv1.CreateProductRequest{
		Name: name, PriceCents: price, CategoryId: cat.GetCategory().GetId(), InitialStock: 3,
	})
	require.NoError(t, err)
	return resp.GetProduct()
}

func TestCreateProduct_ValidatesAndEmitsEvent(t *testing.T) {
	ctx := context.Background()
	s := newService(t)

	_, err := s.CreateProduct(ctx, &productv1.CreateProductRequest{Name: "x", PriceCents: 100, CategoryId: 999})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "unknown category")
	_, err = s.CreateProduct(ctx, &productv1.CreateProductRequest{Name: "x", PriceCents: 0, CategoryId: 1})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "price must be positive")

	createProduct(t, s, "Keyboard", 1999)
	var outbox []events.OutboxMessage
	require.NoError(t, s.db.Find(&outbox).Error)
	require.Len(t, outbox, 1)
	assert.Equal(t, events.TopicProducts, outbox[0].Topic)
}

func TestBatchGetProducts_FailsOnMissingProduct(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := createProduct(t, s, "Mouse", 500)

	resp, err := s.BatchGetProducts(ctx, &productv1.BatchGetProductsRequest{Ids: []uint64{p.GetId(), p.GetId()}})
	require.NoError(t, err)
	assert.Len(t, resp.GetProducts(), 1)

	_, err = s.BatchGetProducts(ctx, &productv1.BatchGetProductsRequest{Ids: []uint64{p.GetId(), 12345}})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestListProducts_SearchAndPagination(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	createProduct(t, s, "Red Keyboard", 100)
	createProduct(t, s, "Blue Keyboard", 100)
	createProduct(t, s, "Mouse", 100)

	resp, err := s.ListProducts(ctx, &productv1.ListProductsRequest{Query: "keyboard", PageSize: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.GetTotal())
	assert.Len(t, resp.GetProducts(), 1)
}

func TestReviewEvents_UpdateRatingOnce(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	p := createProduct(t, s, "Monitor", 20000)

	created, _ := events.New(events.ReviewCreated, events.ReviewData{ReviewID: 1, ProductID: p.GetId(), Rating: 4})
	require.NoError(t, s.HandleReviewEvent(ctx, created))
	require.NoError(t, s.HandleReviewEvent(ctx, created)) // duplicate delivery
	other, _ := events.New(events.ReviewCreated, events.ReviewData{ReviewID: 2, ProductID: p.GetId(), Rating: 5})
	require.NoError(t, s.HandleReviewEvent(ctx, other))

	got, err := s.GetProduct(ctx, &productv1.GetProductRequest{Id: p.GetId()})
	require.NoError(t, err)
	assert.Equal(t, uint32(2), got.GetProduct().GetRatingCount())
	assert.Equal(t, 4.5, got.GetProduct().GetRatingAverage())

	deleted, _ := events.New(events.ReviewDeleted, events.ReviewData{ReviewID: 2, ProductID: p.GetId(), Rating: 5})
	require.NoError(t, s.HandleReviewEvent(ctx, deleted))
	got, err = s.GetProduct(ctx, &productv1.GetProductRequest{Id: p.GetId()})
	require.NoError(t, err)
	assert.Equal(t, 4.0, got.GetProduct().GetRatingAverage())
}
