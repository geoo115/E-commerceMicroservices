package tests

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	"github.com/geoo115/E-commerceMicroservices/review-service/db"
	pb "github.com/geoo115/E-commerceMicroservices/review-service/proto"
	"github.com/geoo115/E-commerceMicroservices/review-service/services"
)

var reviewServer *services.ReviewServer

func TestMain(m *testing.M) {
	// Load .env file from the project root (adjust path if necessary)
	if err := godotenv.Load("../.env"); err != nil {
		// Set default env variables if .env is not found
		os.Setenv("DATABASE_HOST", "localhost")
		os.Setenv("DATABASE_USER", "usr")
		os.Setenv("DATABASE_PASSWORD", "test123")
		os.Setenv("DATABASE_NAME", "ecommerce_users")
		os.Setenv("DATABASE_PORT", "5432")
		os.Setenv("DATABASE_SSLMODE", "disable")
		os.Setenv("REVIEW_SERVICE_PORT", "50056")
	}

	// Initialize the database.
	db.InitDB()

	// Ensure the referenced products table exists and insert a dummy product with id 202.
	// (This is needed to satisfy the foreign key constraint on reviews.product_id.)
	db.DB.Exec(`CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL
	)`)
	// Insert a dummy product with id 202. Use ON CONFLICT to avoid error if it already exists.
	db.DB.Exec(`INSERT INTO products (id, name) VALUES (202, 'Dummy Product') ON CONFLICT (id) DO NOTHING`)

	// Clean up any existing data for a fresh test run.
	db.DB.Exec("DELETE FROM reviews")
	db.DB.Exec("DELETE FROM wishlists")

	reviewServer = services.NewReviewServer()

	code := m.Run()

	db.CloseDB()
	os.Exit(code)
}

func TestCreateAndGetReview(t *testing.T) {
	// Create a review.
	createReq := &pb.CreateReviewRequest{
		UserId:    1,
		ProductId: 202,
		Rating:    4,
		Comment:   "Good product",
	}
	createResp, err := reviewServer.CreateReview(context.Background(), createReq)
	assert.NoError(t, err)
	assert.NotNil(t, createResp)
	assert.Equal(t, uint64(1), createResp.UserId)
	assert.Equal(t, uint64(202), createResp.ProductId)
	assert.Equal(t, int32(4), createResp.Rating)
	assert.Equal(t, "Good product", createResp.Comment)

	// Get the review.
	getReq := &pb.GetReviewRequest{ReviewId: createResp.ReviewId}
	getResp, err := reviewServer.GetReview(context.Background(), getReq)
	assert.NoError(t, err)
	assert.NotNil(t, getResp)
	assert.Equal(t, createResp.ReviewId, getResp.ReviewId)
}

func TestListAndDeleteReview(t *testing.T) {
	// Create two reviews for the same product.
	req1 := &pb.CreateReviewRequest{
		UserId:    2,
		ProductId: 202,
		Rating:    5,
		Comment:   "Excellent!",
	}
	req2 := &pb.CreateReviewRequest{
		UserId:    3,
		ProductId: 202,
		Rating:    3,
		Comment:   "Average product",
	}

	resp1, err := reviewServer.CreateReview(context.Background(), req1)
	assert.NoError(t, err)
	assert.NotNil(t, resp1)

	// Create second review; we don't need to use its response further.
	_, err = reviewServer.CreateReview(context.Background(), req2)
	assert.NoError(t, err)

	// List reviews for product 202.
	listReq := &pb.ListReviewsRequest{ProductId: 202}
	listResp, err := reviewServer.ListReviews(context.Background(), listReq)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)
	// Expect at least two reviews.
	assert.GreaterOrEqual(t, len(listResp.Reviews), 2)

	// Delete one review.
	delReq := &pb.DeleteReviewRequest{ReviewId: resp1.ReviewId}
	delResp, err := reviewServer.DeleteReview(context.Background(), delReq)
	assert.NoError(t, err)
	assert.NotNil(t, delResp)

	// Ensure the deleted review cannot be retrieved.
	getReq := &pb.GetReviewRequest{ReviewId: resp1.ReviewId}
	getResp, err := reviewServer.GetReview(context.Background(), getReq)
	assert.Error(t, err)
	assert.Nil(t, getResp)
}

func TestWishlistOperations(t *testing.T) {
	// Add an item to wishlist.
	addReq := &pb.AddToWishlistRequest{
		UserId:    1,
		ProductId: 303,
	}
	addResp, err := reviewServer.AddToWishlist(context.Background(), addReq)
	assert.NoError(t, err)
	assert.NotNil(t, addResp)

	// Get wishlist for the user.
	getListReq := &pb.GetWishlistRequest{UserId: 1}
	listResp, err := reviewServer.GetWishlist(context.Background(), getListReq)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)
	// Expect at least one wishlist item.
	assert.GreaterOrEqual(t, len(listResp.Items), 1)

	// Remove the item from wishlist.
	remReq := &pb.RemoveFromWishlistRequest{
		UserId:    1,
		ProductId: 303,
	}
	remResp, err := reviewServer.RemoveFromWishlist(context.Background(), remReq)
	assert.NoError(t, err)
	assert.NotNil(t, remResp)

	// Verify removal by getting the wishlist again.
	listRespAfter, err := reviewServer.GetWishlist(context.Background(), getListReq)
	assert.NoError(t, err)
	// The removed item should not be present.
	for _, item := range listRespAfter.Items {
		assert.NotEqual(t, uint64(303), item.ProductId)
	}
}
