package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/geoo115/E-commerceMicroservices/inventory-service/cache"
	"github.com/geoo115/E-commerceMicroservices/inventory-service/db"
	"github.com/geoo115/E-commerceMicroservices/inventory-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/inventory-service/proto"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InventoryServer implements the gRPC InventoryService.
type InventoryServer struct {
	pb.UnimplementedInventoryServiceServer
}

// NewInventoryServer returns a new InventoryServer.
func NewInventoryServer() *InventoryServer {
	return &InventoryServer{}
}

// GetInventory retrieves the current stock for a given product.
func (s *InventoryServer) GetInventory(ctx context.Context, req *pb.GetInventoryRequest) (*pb.InventoryResponse, error) {
	cacheKey := fmt.Sprintf("inventory:%d", req.ProductId)
	// Check Redis cache first.
	cached, err := cache.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var resp pb.InventoryResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
		// If unmarshalling fails, fall back to DB.
	}

	// Query the database if cache miss.
	var inv models.Inventory
	if err := db.DB.Where("product_id = ?", req.ProductId).First(&inv).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("inventory record not found for product %d", req.ProductId)
		}
		return nil, fmt.Errorf("failed to get inventory: %v", err)
	}

	resp := &pb.InventoryResponse{
		ProductId: uint32(inv.ProductID),
		Stock:     int32(inv.Stock),
	}

	// Cache the response for 5 minutes.
	if bytes, err := json.Marshal(resp); err == nil {
		cache.RedisClient.Set(ctx, cacheKey, bytes, 5*time.Minute)
	}

	return resp, nil
}

// UpdateStock updates the inventory stock for a given product.
// The delta value may be positive (restock) or negative (deduction).
func (s *InventoryServer) UpdateStock(ctx context.Context, req *pb.UpdateStockRequest) (*pb.InventoryResponse, error) {
	var inv models.Inventory
	var product models.Product

	// Begin a transaction.
	tx := db.DB.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", tx.Error)
	}

	// Ensure the product exists.
	if err := tx.Where("id = ?", req.ProductId).First(&product).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("product does not exist: %d", req.ProductId)
		}
		return nil, fmt.Errorf("failed to fetch product: %v", err)
	}

	// Lock the inventory row to prevent concurrent modifications.
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("product_id = ?", req.ProductId).
		First(&inv).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create inventory if not found.
			inv = models.Inventory{
				ProductID: uint(req.ProductId),
				Stock:     0,
			}
			if err := tx.Create(&inv).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to create inventory record: %v", err)
			}
		} else {
			tx.Rollback()
			return nil, fmt.Errorf("failed to fetch inventory: %v", err)
		}
	}

	// Calculate the new stock.
	newStock := inv.Stock + int(req.Delta)
	if newStock < 0 {
		tx.Rollback()
		return nil, fmt.Errorf("insufficient stock for product %d", req.ProductId)
	}

	// Update the stock.
	if err := tx.Model(&inv).Update("stock", newStock).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update stock: %v", err)
	}

	// Commit the transaction.
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Flush the cached inventory.
	cacheKey := fmt.Sprintf("inventory:%d", req.ProductId)
	cache.RedisClient.Del(ctx, cacheKey)

	log.Printf("Product %d stock updated from %d to %d", req.ProductId, inv.Stock, newStock)
	return &pb.InventoryResponse{
		ProductId: req.ProductId,
		Stock:     int32(newStock),
	}, nil
}
