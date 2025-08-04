package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/geoo115/E-commerceMicroservices/message-broker/producers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/topics"
	"github.com/geoo115/E-commerceMicroservices/order-service/db"
	"github.com/geoo115/E-commerceMicroservices/order-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/order-service/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OrderServer implements the gRPC OrderService.
type OrderServer struct {
	pb.UnimplementedOrderServiceServer
}

// NewOrderServer creates a new instance of OrderServer.
func NewOrderServer() *OrderServer {
	return &OrderServer{}
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += item.Price * float64(item.Quantity)
	}

	order := models.Order{
		UserID:      uint(req.UserId),
		TotalAmount: totalAmount,
		Status:      "created",
	}

	tx := db.DB.Begin()
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create order: %v", err)
	}
	for _, item := range req.Items {
		orderItem := models.OrderItem{
			OrderID:   order.ID,
			ProductID: uint(item.ProductId),
			Quantity:  int(item.Quantity),
			Price:     item.Price,
		}
		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create order item: %v", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("transaction failed: %v", err)
	}

	// Publish order_placed event.
	eventPayload, err := json.Marshal(order)
	if err != nil {
		log.Printf("Failed to marshal order event: %v", err)
	} else {
		if err := producers.PublishEvent(topics.OrderPlaced, eventPayload); err != nil {
			log.Printf("Failed to publish order event: %v", err)
		}
	}

	return s.getOrderResponse(order.ID)
}

// getOrderResponse constructs an OrderResponse based on the order ID.
func (s *OrderServer) getOrderResponse(id uint) (*pb.OrderResponse, error) {
	var order models.Order
	if err := db.DB.Preload("Items").First(&order, id).Error; err != nil {
		return nil, fmt.Errorf("order not found: %v", err)
	}

	resp := &pb.OrderResponse{
		OrderId:     uint64(order.ID),
		UserId:      uint64(order.UserID),
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
	}
	// Convert order items.
	for _, item := range order.Items {
		resp.Items = append(resp.Items, &pb.OrderItemResponse{
			OrderItemId: uint64(item.ID),
			ProductId:   uint64(item.ProductID),
			Quantity:    int32(item.Quantity),
			Price:       item.Price,
		})
	}
	return resp, nil
}

// UpdateOrderStatus updates the status of an existing order.
func (s *OrderServer) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.OrderResponse, error) {
	var order models.Order
	if err := db.DB.First(&order, req.OrderId).Error; err != nil {
		return nil, fmt.Errorf("order not found: %v", err)
	}

	order.Status = req.Status

	if err := db.DB.Save(&order).Error; err != nil {
		return nil, fmt.Errorf("failed to update order status: %v", err)
	}

	return s.getOrderResponse(order.ID)
}

func (s *OrderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	var order models.Order
	orderID := uint(req.OrderId)

	if err := db.DB.Preload("Items").
		Where("id = ?", orderID).
		First(&order).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "order not found")
	}

	return convertToOrderResponse(order), nil
}

func convertToOrderResponse(o models.Order) *pb.OrderResponse {
	items := make([]*pb.OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = &pb.OrderItemResponse{
			OrderItemId: uint64(item.ID),
			ProductId:   uint64(item.ProductID),
			Quantity:    int32(item.Quantity),
			Price:       item.Price,
		}
	}

	return &pb.OrderResponse{
		OrderId:     uint64(o.ID),
		UserId:      uint64(o.UserID),
		TotalAmount: o.TotalAmount,
		Status:      o.Status,
		Items:       items,
	}
}

// HandlePaymentSuccessfulEvent processes payment successful events
func HandlePaymentSuccessfulEvent(message []byte) {
	// TODO: Implement payment successful event handling
	// This could involve updating order status, sending notifications, etc.
	log.Printf("Processing payment successful event: %s", string(message))
}
