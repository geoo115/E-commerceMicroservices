package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	"payment-service/db"
	pb "payment-service/proto"
	"payment-service/services"
)

var paymentServer *services.PaymentServer

func TestMain(m *testing.M) {
	// Load environment variables from .env file (adjust the path if necessary)
	if err := godotenv.Load("../.env"); err != nil {
		// If .env not found, set defaults for testing.
		os.Setenv("DATABASE_HOST", "localhost")
		os.Setenv("DATABASE_USER", "usr")
		os.Setenv("DATABASE_PASSWORD", "test123")
		os.Setenv("DATABASE_NAME", "ecommerce_users")
		os.Setenv("DATABASE_PORT", "5432")
		os.Setenv("DATABASE_SSLMODE", "disable")
		os.Setenv("PAYMENT_SERVICE_PORT", "50055")
	}

	// Initialize the database
	db.InitDB()

	// Clean up Payment records if necessary. (For a clean test run, you might delete previous payments)
	db.DB.Exec("DELETE FROM payments")

	// Initialize PaymentServer instance
	paymentServer = services.NewPaymentServer()

	// Run tests
	code := m.Run()

	// Close DB connection and exit
	db.CloseDB()
	os.Exit(code)
}

func TestProcessPayment(t *testing.T) {
	req := &pb.ProcessPaymentRequest{
		OrderId:       1001,
		Amount:        150.75,
		Currency:      "USD",
		PaymentMethod: "Credit Card",
		CardLastFour:  "1234",
	}

	resp, err := paymentServer.ProcessPayment(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify that the returned payment has the expected fields.
	assert.Equal(t, req.OrderId, resp.OrderId)
	assert.Equal(t, req.Amount, resp.Amount)
	assert.Equal(t, req.Currency, resp.Currency)
	assert.Equal(t, req.PaymentMethod, resp.PaymentMethod)
	assert.Equal(t, "Success", resp.Status) // In our simulated processing, we mark it as Success.
	assert.NotEmpty(t, resp.TransactionId)
	// Check that processed_at is a valid timestamp.
	_, err = time.Parse(time.RFC3339, resp.ProcessedAt)
	assert.NoError(t, err)
}

func TestGetPayment(t *testing.T) {
	// First, process a payment so we have a record.
	req := &pb.ProcessPaymentRequest{
		OrderId:       2002,
		Amount:        250.00,
		Currency:      "USD",
		PaymentMethod: "PayPal",
		CardLastFour:  "5678",
	}
	processResp, err := paymentServer.ProcessPayment(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, processResp)

	// Now retrieve the payment using its ID.
	getReq := &pb.GetPaymentRequest{
		PaymentId: processResp.PaymentId,
	}
	getResp, err := paymentServer.GetPayment(context.Background(), getReq)
	assert.NoError(t, err)
	assert.NotNil(t, getResp)

	// Verify the fields match.
	assert.Equal(t, processResp.PaymentId, getResp.PaymentId)
	assert.Equal(t, processResp.OrderId, getResp.OrderId)
	assert.Equal(t, processResp.Amount, getResp.Amount)
	assert.Equal(t, processResp.Currency, getResp.Currency)
	assert.Equal(t, processResp.PaymentMethod, getResp.PaymentMethod)
	assert.Equal(t, processResp.Status, getResp.Status)
}
