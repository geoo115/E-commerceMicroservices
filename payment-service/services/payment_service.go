package services

import (
	"context"
	"fmt"
	"time"

	"github.com/geoo115/E-commerceMicroservices/payment-service/db"
	"github.com/geoo115/E-commerceMicroservices/payment-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/payment-service/proto"

	"github.com/google/uuid"
)

// PaymentServer implements the gRPC PaymentService.
type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
}

// NewPaymentServer returns a new PaymentServer.
func NewPaymentServer() *PaymentServer {
	return &PaymentServer{}
}

// ProcessPayment processes a payment and stores it in the database.
func (s *PaymentServer) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.PaymentResponse, error) {
	// Create a Payment record.
	payment := models.Payment{
		OrderID:         uint(req.OrderId),
		TransactionID:   uuid.NewString(), // Generate a unique transaction ID.
		PaymentMethod:   models.PaymentMethod(req.PaymentMethod),
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          models.PaymentPending, // Initial status.
		ProcessedAt:     time.Now(),
		CardLastFour:    req.CardLastFour,
		PaymentGateway:  "TestGateway",
		GatewayResponse: "Payment processing simulated",
	}

	// Validate payment before saving.
	if err := payment.Validate(); err != nil {
		return nil, fmt.Errorf("payment validation failed: %v", err)
	}

	if err := db.DB.Create(&payment).Error; err != nil {
		return nil, fmt.Errorf("failed to process payment: %v", err)
	}

	// For testing purposes, simulate success.
	payment.Status = models.PaymentSuccess
	db.DB.Save(&payment)

	return &pb.PaymentResponse{
		PaymentId:       uint64(payment.ID),
		OrderId:         uint64(payment.OrderID),
		Amount:          payment.Amount,
		Currency:        payment.Currency,
		PaymentMethod:   string(payment.PaymentMethod),
		Status:          string(payment.Status),
		TransactionId:   payment.TransactionID,
		FailureReason:   payment.FailureReason,
		PaymentGateway:  payment.PaymentGateway,
		GatewayResponse: payment.GatewayResponse,
		ProcessedAt:     payment.ProcessedAt.Format(time.RFC3339),
	}, nil
}

// GetPayment retrieves a payment by its ID.
func (s *PaymentServer) GetPayment(ctx context.Context, req *pb.GetPaymentRequest) (*pb.PaymentResponse, error) {
	var payment models.Payment
	if err := db.DB.First(&payment, req.PaymentId).Error; err != nil {
		return nil, fmt.Errorf("payment not found: %v", err)
	}

	return &pb.PaymentResponse{
		PaymentId:       uint64(payment.ID),
		OrderId:         uint64(payment.OrderID),
		Amount:          payment.Amount,
		Currency:        payment.Currency,
		PaymentMethod:   string(payment.PaymentMethod),
		Status:          string(payment.Status),
		TransactionId:   payment.TransactionID,
		FailureReason:   payment.FailureReason,
		PaymentGateway:  payment.PaymentGateway,
		GatewayResponse: payment.GatewayResponse,
		ProcessedAt:     payment.ProcessedAt.Format(time.RFC3339),
	}, nil
}
