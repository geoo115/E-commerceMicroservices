package tests

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	"cart-service/db"
	pb "cart-service/proto"
	"cart-service/services"
)

var cartServer *services.CartServer

// TestMain sets up the environment, initializes the database, and creates the CartServer instance.
func TestMain(m *testing.M) {
	// Attempt to load .env file from the project root (adjust path as needed)
	if err := godotenv.Load("../.env"); err != nil {
		// If .env is not found, set defaults for testing
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_USER", "usr")
		os.Setenv("DB_PASSWORD", "test123")
		os.Setenv("DB_NAME", "ecommerce_users")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("CART_SERVICE_PORT", "50054")
	}

	// Initialize the database
	db.InitDB()
	// Clean up any existing cart data for a fresh test run
	cleanupCartDatabase()

	// Initialize the CartServer instance
	cartServer = services.NewCartServer()

	// Run tests
	code := m.Run()

	// Close database connection
	db.CloseDB()

	os.Exit(code)
}

// cleanupCartDatabase clears the carts table.
func cleanupCartDatabase() {
	db.DB.Exec("DELETE FROM carts")
}

// TestAddItemToCart tests the AddItemToCart RPC method.
func TestAddItemToCart(t *testing.T) {
	req := &pb.AddItemRequest{
		UserId:    1,
		ProductId: 101,
		Quantity:  2,
	}
	resp, err := cartServer.AddItemToCart(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, uint64(1), resp.UserId)
	assert.Equal(t, uint64(101), resp.ProductId)
	assert.Equal(t, int32(2), resp.Quantity)
}

// TestUpdateCartItem tests updating the quantity of an existing cart item.
func TestUpdateCartItem(t *testing.T) {
	// First, add an item
	req := &pb.AddItemRequest{
		UserId:    1,
		ProductId: 102,
		Quantity:  1,
	}
	_, err := cartServer.AddItemToCart(context.Background(), req)
	assert.NoError(t, err)

	// Update the item quantity
	updateReq := &pb.UpdateItemRequest{
		UserId:    1,
		ProductId: 102,
		Quantity:  3,
	}
	updateResp, err := cartServer.UpdateCartItem(context.Background(), updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updateResp)
	assert.Equal(t, int32(3), updateResp.Quantity)
}

// TestRemoveCartItem tests removing an item from the cart.
func TestRemoveCartItem(t *testing.T) {
	// First, add an item
	req := &pb.AddItemRequest{
		UserId:    1,
		ProductId: 103,
		Quantity:  1,
	}
	_, err := cartServer.AddItemToCart(context.Background(), req)
	assert.NoError(t, err)

	// Remove the item
	removeReq := &pb.RemoveItemRequest{
		UserId:    1,
		ProductId: 103,
	}
	removeResp, err := cartServer.RemoveCartItem(context.Background(), removeReq)
	assert.NoError(t, err)
	assert.NotNil(t, removeResp)
	// After removal, quantity should be 0
	assert.Equal(t, int32(0), removeResp.Quantity)
}

// TestGetCart tests retrieving all cart items for a given user.
func TestGetCart(t *testing.T) {
	// Clean up any existing cart items for user 2.
	db.DB.Exec("DELETE FROM carts WHERE user_id = ?", 2)

	// Add two items for user 2.
	item1 := &pb.AddItemRequest{
		UserId:    2,
		ProductId: 201,
		Quantity:  2,
	}
	item2 := &pb.AddItemRequest{
		UserId:    2,
		ProductId: 202,
		Quantity:  5,
	}
	_, err := cartServer.AddItemToCart(context.Background(), item1)
	assert.NoError(t, err)
	_, err = cartServer.AddItemToCart(context.Background(), item2)
	assert.NoError(t, err)

	// Retrieve the cart for user 2.
	getReq := &pb.GetCartRequest{
		UserId: 2,
	}
	getResp, err := cartServer.GetCart(context.Background(), getReq)
	assert.NoError(t, err)
	assert.NotNil(t, getResp)
	// Expect at least two items in the cart.
	assert.GreaterOrEqual(t, len(getResp.Items), 2)
}
