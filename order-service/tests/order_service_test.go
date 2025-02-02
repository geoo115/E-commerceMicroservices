package tests

import (
	"context"
	"order-service/db"
	pb "order-service/proto"
	"order-service/services"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

var orderServer *services.OrderServer

// TestMain loads the environment variables, initializes the database,
// clears any pre-existing test data, and creates the OrderServer instance.
func TestMain(m *testing.M) {
	// Load .env file from the project root or fallback to default env settings.
	if err := godotenv.Load("../.env"); err != nil {
		// If .env not found, manually set required environment variables.
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_USER", "your_db_user")
		os.Setenv("DB_PASSWORD", "your_db_password")
		os.Setenv("DB_NAME", "your_db_name")
		os.Setenv("DB_PORT", "5432")
		os.Setenv("ORDER_SERVICE_PORT", "50053")
	}

	// Initialize the database connection.
	db.InitDB()

	// Clean up any existing data (for test isolation).
	cleanupDatabase()

	// Initialize the OrderServer instance.
	orderServer = services.NewOrderServer()

	// Run the tests.
	code := m.Run()

	// Close the database connection.
	db.CloseDB()

	os.Exit(code)
}

// cleanupDatabase deletes data from orders and order_items tables.
func cleanupDatabase() {
	// Delete order items first, then orders.
	db.DB.Exec("DELETE FROM order_items")
	db.DB.Exec("DELETE FROM orders")
}

// Helper: createTestOrder creates an order via the service and returns the response.
func createTestOrder(t *testing.T, userID uint64, items []*pb.OrderItemRequest) *pb.OrderResponse {
	req := &pb.CreateOrderRequest{
		UserId: userID,
		Items:  items,
	}
	resp, err := orderServer.CreateOrder(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	return resp
}

// TestCreateOrder verifies that an order can be created successfully.
func TestCreateOrder(t *testing.T) {
	// Create a test order with one order item.
	item := &pb.OrderItemRequest{
		ProductId: 101,
		Quantity:  2,
		Price:     9.99,
	}
	resp := createTestOrder(t, 1, []*pb.OrderItemRequest{item})

	// Verify the response values.
	assert.Equal(t, uint64(1), resp.UserId)
	assert.Equal(t, "created", resp.Status)
	assert.Equal(t, float64(2*9.99), resp.TotalAmount)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, uint64(101), resp.Items[0].ProductId)
	assert.Equal(t, int32(2), resp.Items[0].Quantity)
}

// TestGetOrder verifies that an existing order can be retrieved.
func TestGetOrder(t *testing.T) {
	// Create a test order.
	item := &pb.OrderItemRequest{
		ProductId: 202,
		Quantity:  1,
		Price:     19.99,
	}
	createResp := createTestOrder(t, 2, []*pb.OrderItemRequest{item})

	// Retrieve the order using GetOrder.
	getReq := &pb.GetOrderRequest{
		OrderId: createResp.OrderId,
	}
	getResp, err := orderServer.GetOrder(context.Background(), getReq)
	assert.NoError(t, err)
	assert.NotNil(t, getResp)
	assert.Equal(t, createResp.OrderId, getResp.OrderId)
	assert.Equal(t, uint64(2), getResp.UserId)
	assert.Equal(t, createResp.TotalAmount, getResp.TotalAmount)
}

// TestListOrders verifies that orders can be listed for a given user.
func TestListOrders(t *testing.T) {
	// Create multiple orders for user 3.
	for i := 0; i < 3; i++ {
		item := &pb.OrderItemRequest{
			ProductId: uint64(300 + i),
			Quantity:  1 + int32(i),
			Price:     10.0 + float64(i),
		}
		createTestOrder(t, 3, []*pb.OrderItemRequest{item})
	}

	// List orders for user 3.
	listReq := &pb.ListOrdersRequest{
		UserId: 3,
		Page:   1,
		Limit:  10,
	}
	listResp, err := orderServer.ListOrders(context.Background(), listReq)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)
	assert.GreaterOrEqual(t, len(listResp.Orders), 3)
}

// TestDeleteOrder verifies that an order can be deleted.
func TestDeleteOrder(t *testing.T) {
	// Create a test order.
	item := &pb.OrderItemRequest{
		ProductId: 404,
		Quantity:  1,
		Price:     29.99,
	}
	createResp := createTestOrder(t, 4, []*pb.OrderItemRequest{item})

	// Delete the order.
	delReq := &pb.GetOrderRequest{OrderId: createResp.OrderId}
	delResp, err := orderServer.DeleteOrder(context.Background(), delReq)
	assert.NoError(t, err)
	assert.NotNil(t, delResp)
	assert.Equal(t, createResp.OrderId, delResp.OrderId)

	// Verify the order no longer exists.
	getReq := &pb.GetOrderRequest{OrderId: createResp.OrderId}
	getResp, err := orderServer.GetOrder(context.Background(), getReq)
	// Expect an error since the order should be deleted.
	assert.Error(t, err)
	assert.Nil(t, getResp)
}

// TestUpdateOrderStatus verifies that an order’s status can be updated.
func TestUpdateOrderStatus(t *testing.T) {
	// Create a test order.
	item := &pb.OrderItemRequest{
		ProductId: 505,
		Quantity:  3,
		Price:     15.0,
	}
	createResp := createTestOrder(t, 5, []*pb.OrderItemRequest{item})

	// Update the order's status.
	updateReq := &pb.UpdateOrderStatusRequest{
		OrderId: createResp.OrderId,
		Status:  "shipped",
	}
	updateResp, err := orderServer.UpdateOrderStatus(context.Background(), updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updateResp)
	assert.Equal(t, "shipped", updateResp.Status)
}
