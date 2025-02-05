package services

import (
	"context"
	"fmt"
	"log"

	"github.com/geoo115/E-commerceMicroservices/order-service/db"
	"github.com/geoo115/E-commerceMicroservices/order-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/order-service/proto"
)

// OrderServer implements the gRPC OrderService.
type OrderServer struct {
	pb.UnimplementedOrderServiceServer
}

// NewOrderServer creates and returns a new OrderServer.
func NewOrderServer() *OrderServer {
	return &OrderServer{}
}

// CreateOrder creates a new order along with its order items.
func (s *OrderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	// Calculate the total amount.
	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += item.Price * float64(item.Quantity)
	}

	order := models.Order{
		UserID:      uint(req.UserId),
		TotalAmount: totalAmount,
		Status:      "created",
	}

	// Begin a transaction.
	tx := db.DB.Begin()
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// Create each order item.
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
		tx.Rollback()
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return s.getOrderResponse(order.ID)
}

// GetOrder retrieves an order by its ID.
func (s *OrderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	return s.getOrderResponse(uint(req.OrderId))
}

// ListOrders lists orders for a given user with pagination.
func (s *OrderServer) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	var orders []models.Order
	offset := (req.Page - 1) * req.Limit

	if err := db.DB.Where("user_id = ?", req.UserId).Offset(int(offset)).Limit(int(req.Limit)).Find(&orders).Error; err != nil {
		return nil, fmt.Errorf("failed to list orders: %v", err)
	}

	var responses []*pb.OrderResponse
	for _, order := range orders {
		resp, err := s.getOrderResponse(order.ID)
		if err != nil {
			log.Printf("Error fetching order response for order ID %d: %v", order.ID, err)
			continue
		}
		responses = append(responses, resp)
	}

	return &pb.ListOrdersResponse{Orders: responses}, nil
}

// DeleteOrder deletes an order by its ID.
// We use the same request message as GetOrderRequest.
func (s *OrderServer) DeleteOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	// First, retrieve the order so we can return its data.
	orderResp, err := s.getOrderResponse(uint(req.OrderId))
	if err != nil {
		return nil, err
	}

	// Delete the order items.
	if err := db.DB.Where("order_id = ?", req.OrderId).Delete(&models.OrderItem{}).Error; err != nil {
		return nil, fmt.Errorf("failed to delete order items: %v", err)
	}

	// Delete the order.
	if err := db.DB.Delete(&models.Order{}, req.OrderId).Error; err != nil {
		return nil, fmt.Errorf("failed to delete order: %v", err)
	}

	// Return the data of the deleted order.
	return orderResp, nil
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

// getOrderResponse is a helper that builds an OrderResponse for a given order ID.
func (s *OrderServer) getOrderResponse(orderID uint) (*pb.OrderResponse, error) {
	var order models.Order
	if err := db.DB.First(&order, orderID).Error; err != nil {
		return nil, fmt.Errorf("order not found: %v", err)
	}

	// Retrieve order items.
	var orderItems []models.OrderItem
	if err := db.DB.Where("order_id = ?", order.ID).Find(&orderItems).Error; err != nil {
		return nil, fmt.Errorf("failed to get order items: %v", err)
	}

	// Build response.
	resp := &pb.OrderResponse{
		OrderId:     uint64(order.ID),
		UserId:      uint64(order.UserID),
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
	}

	for _, item := range orderItems {
		resp.Items = append(resp.Items, &pb.OrderItemResponse{
			OrderItemId: uint64(item.ID),
			ProductId:   uint64(item.ProductID),
			Quantity:    int32(item.Quantity),
			Price:       item.Price,
		})
	}

	return resp, nil
}
