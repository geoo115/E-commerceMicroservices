package tests

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/geoo115/E-commerceMicroservices/inventory-service/db"
	"github.com/geoo115/E-commerceMicroservices/inventory-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/inventory-service/proto"
	"github.com/geoo115/E-commerceMicroservices/inventory-service/services"
)

var inventoryServer *services.InventoryServer

func TestMain(m *testing.M) {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		setDefaultEnv()
	}

	// Initialize database
	db.InitDB()
	defer db.CloseDB()

	// Create test data
	if err := createTestData(); err != nil {
		panic("failed to create test data: " + err.Error())
	}

	inventoryServer = services.NewInventoryServer()
	code := m.Run()

	// Cleanup
	cleanupTestData()
	os.Exit(code)
}

func setDefaultEnv() {
	envVars := map[string]string{
		"DATABASE_HOST":     "localhost",
		"DATABASE_USER":     "usr",
		"DATABASE_PASSWORD": "test123",
		"DATABASE_NAME":     "ecommerce",
		"DATABASE_PORT":     "5432",
		"DATABASE_SSLMODE":  "disable",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}
}

func createTestData() error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		// Ensure category exists
		category := models.Category{Model: gorm.Model{ID: 1}, Name: "Test Category"}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&category).Error; err != nil {
			return err
		}

		// Ensure products exist
		products := []models.Product{
			{Model: gorm.Model{ID: 1001}, Name: "Test Product 1", Price: 10.0, CategoryID: 1},
			{Model: gorm.Model{ID: 1002}, Name: "Test Product 2", Price: 20.0, CategoryID: 1},
		}

		for _, p := range products {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&p).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func cleanupTestData() {
	// Delete dependent records first
	db.DB.Exec("DELETE FROM reviews")     // Depends on products
	db.DB.Exec("DELETE FROM order_items") // Depends on orders
	db.DB.Exec("DELETE FROM orders")      // Depends on products
	db.DB.Exec("DELETE FROM inventories") // Depends on products
	db.DB.Exec("DELETE FROM products")    // Depends on categories
	db.DB.Exec("DELETE FROM categories")  // No dependencies left
}

func TestInventoryService(t *testing.T) {
	t.Run("Basic stock operations", func(t *testing.T) {
		productID := uint32(1001)
		ctx := context.Background()

		// Initial update
		updateResp, err := inventoryServer.UpdateStock(ctx, &pb.UpdateStockRequest{
			ProductId: productID,
			Delta:     50,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(50), updateResp.Stock)

		// Deduct stock
		updateResp, err = inventoryServer.UpdateStock(ctx, &pb.UpdateStockRequest{
			ProductId: productID,
			Delta:     -20,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(30), updateResp.Stock)

		// Verify
		getResp, err := inventoryServer.GetInventory(ctx, &pb.GetInventoryRequest{
			ProductId: productID,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(30), getResp.Stock)
	})

	t.Run("Prevent negative stock", func(t *testing.T) {
		productID := uint32(1002)
		ctx := context.Background()

		// Setup initial stock
		_, err := inventoryServer.UpdateStock(ctx, &pb.UpdateStockRequest{
			ProductId: productID,
			Delta:     30,
		})
		assert.NoError(t, err)

		// Test over-deduction
		_, err = inventoryServer.UpdateStock(ctx, &pb.UpdateStockRequest{
			ProductId: productID,
			Delta:     -40,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient stock")

		// Verify no change
		getResp, err := inventoryServer.GetInventory(ctx, &pb.GetInventoryRequest{
			ProductId: productID,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(30), getResp.Stock)
	})

	t.Run("Handle non-existent product", func(t *testing.T) {
		nonExistentID := uint32(9999)
		ctx := context.Background()

		// Update stock
		_, err := inventoryServer.UpdateStock(ctx, &pb.UpdateStockRequest{
			ProductId: nonExistentID,
			Delta:     10,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "product does not exist")

		// Get inventory
		_, err = inventoryServer.GetInventory(ctx, &pb.GetInventoryRequest{
			ProductId: nonExistentID,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Concurrent updates", func(t *testing.T) {
		productID := uint32(1001)
		const iterations = 50
		var wg sync.WaitGroup
		wg.Add(iterations)

		// Reset stock
		db.DB.Model(&models.Inventory{}).
			Where("product_id = ?", productID).
			Update("stock", 0)

		// Concurrent updates
		for i := 0; i < iterations; i++ {
			go func() {
				defer wg.Done()
				_, err := inventoryServer.UpdateStock(context.Background(), &pb.UpdateStockRequest{
					ProductId: productID,
					Delta:     1,
				})
				assert.NoError(t, err)
			}()
		}

		wg.Wait()

		// Verify final stock
		getResp, err := inventoryServer.GetInventory(context.Background(), &pb.GetInventoryRequest{
			ProductId: productID,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(iterations), getResp.Stock)
	})

	t.Run("Auto-create inventory for new product", func(t *testing.T) {
		// Create new product
		newProduct := models.Product{
			Name:       "New Product",
			Price:      15.99,
			CategoryID: 1,
		}
		err := db.DB.Create(&newProduct).Error
		assert.NoError(t, err)
		defer db.DB.Delete(&newProduct)

		// Test inventory creation
		updateResp, err := inventoryServer.UpdateStock(context.Background(), &pb.UpdateStockRequest{
			ProductId: uint32(newProduct.ID),
			Delta:     100,
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(100), updateResp.Stock)

		// Verify
		getResp, err := inventoryServer.GetInventory(context.Background(), &pb.GetInventoryRequest{
			ProductId: uint32(newProduct.ID),
		})
		assert.NoError(t, err)
		assert.Equal(t, int32(100), getResp.Stock)
	})
}
