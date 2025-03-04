package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/geoo115/E-commerceMicroservices/inventory-service/db"
	"github.com/geoo115/E-commerceMicroservices/inventory-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/inventory-service/proto"
	"github.com/geoo115/E-commerceMicroservices/message-broker/consumers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/producers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/topics"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Define the OrderPlaced topic for consumption.
const OrderPlacedTopic = "order_placed"

// InventoryServer implements the gRPC InventoryService.
type InventoryServer struct {
	pb.UnimplementedInventoryServiceServer
}

// NewInventoryServer returns a new InventoryServer instance.
func NewInventoryServer() *InventoryServer {
	return &InventoryServer{}
}

// GetInventory retrieves the current stock for a given product.
func (s *InventoryServer) GetInventory(ctx context.Context, req *pb.GetInventoryRequest) (*pb.InventoryResponse, error) {
	var inv models.Inventory
	if err := db.DB.Where("product_id = ?", req.ProductId).First(&inv).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("inventory record not found for product %d", req.ProductId)
		}
		return nil, fmt.Errorf("failed to get inventory: %v", err)
	}
	return &pb.InventoryResponse{
		ProductId: uint32(inv.ProductID),
		Stock:     int32(inv.Stock),
	}, nil
}

func (s *InventoryServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.InventoryResponse, error) {
	var inv models.Inventory
	var product models.Product

	tx := db.DB.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", tx.Error)
	}

	// Verify product exists.
	if err := tx.Where("id = ?", req.ProductId).First(&product).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("product does not exist: %d", req.ProductId)
		}
		return nil, fmt.Errorf("failed to fetch product: %v", err)
	}

	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("product_id = ?", req.ProductId).
		First(&inv).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			inv = models.Inventory{ProductID: uint(req.ProductId), Stock: 0}
			if err := tx.Create(&inv).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to create inventory record: %v", err)
			}
		} else {
			tx.Rollback()
			return nil, fmt.Errorf("failed to fetch inventory: %v", err)
		}
	}

	newStock := inv.Stock + int(req.Delta)
	if newStock < 0 {
		tx.Rollback()
		return nil, fmt.Errorf("insufficient stock for product %d", req.ProductId)
	}
	if err := tx.Model(&inv).Update("stock", newStock).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update stock: %v", err)
	}
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Product %d stock updated from %d to %d", req.ProductId, inv.Stock, newStock)

	// Publish inventory_updated event.
	eventData := map[string]interface{}{
		"productId": req.ProductId,
		"stock":     newStock,
	}
	eventPayload, err := json.Marshal(eventData)
	if err != nil {
		log.Printf("Failed to marshal inventory event: %v", err)
	} else {
		if err := producers.PublishEvent(topics.InventoryUpdated, eventPayload); err != nil {
			log.Printf("Failed to publish inventory event: %v", err)
		} else {
			log.Printf("✅ Inventory event published for product %d", req.ProductId)
		}
	}

	return &pb.InventoryResponse{ProductId: req.ProductId, Stock: int32(newStock)}, nil
}

// StartOrderPlacedConsumer listens for order_placed events and adjusts inventory accordingly.
func StartOrderPlacedConsumer() {
	handler := func(message []byte) {
		log.Printf("Inventory Service received order event: %s", string(message))
		// Unmarshal the order event payload.
		var order models.Order
		if err := json.Unmarshal(message, &order); err != nil {
			log.Printf("Failed to unmarshal order event: %v", err)
			return
		}

		// For each order item, deduct the corresponding quantity.
		for _, item := range order.Items {
			log.Printf("Deducting %d from inventory for product %d", item.Quantity, item.ProductID)
			req := &pb.UpdateStockRequest{
				ProductId: uint32(item.ProductID),
				Delta:     -int32(item.Quantity),
			}
			// Use a background context; in production, add proper error handling and retries.
			_, err := NewInventoryServer().UpdateStock(context.Background(), req)
			if err != nil {
				log.Printf("Failed to update stock for product %d: %v", item.ProductID, err)
			}
		}
	}

	// Start Kafka consumer for order_placed events.
	go consumers.ConsumeEvents(OrderPlacedTopic, handler)
}
