package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/geoo115/E-commerceMicroservices/message-broker/producers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/topics"
	"github.com/geoo115/E-commerceMicroservices/payment-service/db"
	"github.com/geoo115/E-commerceMicroservices/payment-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/payment-service/proto"
)

// PaymentServer implements the gRPC PaymentService.
type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
}

// NewPaymentServer returns a new PaymentServer instance.
func NewPaymentServer() *PaymentServer {
	return &PaymentServer{}
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.PaymentResponse, error) {
	payment := models.Payment{
		OrderID:         uint(req.OrderId),
		TransactionID:   uuid.NewString(),
		PaymentMethod:   models.PaymentMethod(req.PaymentMethod),
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          models.PaymentPending,
		ProcessedAt:     time.Now(),
		CardLastFour:    req.CardLastFour,
		PaymentGateway:  "TestGateway",
		GatewayResponse: "Payment processing simulated",
	}
	if err := payment.Validate(); err != nil {
		return nil, fmt.Errorf("payment validation failed: %v", err)
	}
	if err := db.DB.Create(&payment).Error; err != nil {
		return nil, fmt.Errorf("failed to process payment: %v", err)
	}

	// Simulate payment success.
	payment.Status = models.PaymentSuccess
	if err := db.DB.Save(&payment).Error; err != nil {
		return nil, fmt.Errorf("failed to update payment status: %v", err)
	}

	eventPayload, err := json.Marshal(payment)
	if err != nil {
		log.Printf("Failed to marshal payment event: %v", err)
	} else {
		if err := producers.PublishEvent(topics.PaymentSuccessful, eventPayload); err != nil {
			log.Printf("Failed to publish payment event: %v", err)
		} else {
			log.Printf("✅ Payment event published for payment ID %d", payment.ID)
		}
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

// GetPayment retrieves a payment record by its ID.
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
