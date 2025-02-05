package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	"github.com/geoo115/E-commerceMicroservices/product-service/db"
	"github.com/geoo115/E-commerceMicroservices/product-service/models"
	pb "github.com/geoo115/E-commerceMicroservices/product-service/product-service/proto"
	"github.com/geoo115/E-commerceMicroservices/product-service/services"
)

var productServer *services.ProductServer

// TestMain loads environment variables, initializes the DB, and clears the tables.
func TestMain(m *testing.M) {
	// Attempt to load .env file from the project root (adjust path if needed)
	if err := godotenv.Load("../.env"); err != nil {
		// Fallback: manually set required env variables if .env file is not found
		os.Setenv("DATABASE_HOST", "localhost")
		os.Setenv("DATABASE_USER", "usr")
		os.Setenv("DATABASE_PASSWORD", "test123")
		os.Setenv("DATABASE_NAME", "ecommerce_users")
		os.Setenv("DATABASE_PORT", "5432")
		os.Setenv("DATABASE_SSLMODE", "disable")
		os.Setenv("PRODUCT_SERVICE_PORT", "50052")
	}

	// Initialize the database
	db.InitDB()

	// Clear existing data (be cautious: this deletes data from the tables)
	db.DB.Exec("DELETE FROM reviews")
	db.DB.Exec("DELETE FROM inventories")
	db.DB.Exec("DELETE FROM products")
	db.DB.Exec("DELETE FROM categories")

	// Initialize the ProductServer instance
	productServer = services.NewProductServer()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// createTestCategory inserts a new Category in the database for testing purposes.
func createTestCategory(t *testing.T, name string) models.Category {
	category := models.Category{Name: name}
	err := db.DB.Create(&category).Error
	assert.NoError(t, err)
	return category
}

// TestCreateProduct tests the CreateProduct RPC method.
func TestCreateProduct(t *testing.T) {
	category := createTestCategory(t, "Electronics")
	req := &pb.CreateProductRequest{
		Name:        "Test Product",
		Price:       99.99,
		CategoryId:  uint32(category.ID),
		Description: "A test product",
		Stock:       50,
	}
	resp, err := productServer.CreateProduct(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, req.Name, resp.Name)
	assert.Equal(t, req.Price, resp.Price)
	assert.Equal(t, req.Description, resp.Description)
	// Check that the returned Category matches the one we created
	assert.Equal(t, category.ID, uint(resp.Category.Id))
	// Check inventory
	assert.Equal(t, int32(50), resp.Inventory.Stock)
}

// TestGetProduct tests retrieving a product by its ID.
func TestGetProduct(t *testing.T) {
	category := createTestCategory(t, "Books")
	// First, create a product to retrieve
	createReq := &pb.CreateProductRequest{
		Name:        "Test Book",
		Price:       19.99,
		CategoryId:  uint32(category.ID),
		Description: "A test book",
		Stock:       30,
	}
	createResp, err := productServer.CreateProduct(context.Background(), createReq)
	assert.NoError(t, err)
	assert.NotNil(t, createResp)

	// Now, call GetProduct using the created product's ID
	getReq := &pb.GetProductRequest{Id: createResp.Id}
	getResp, err := productServer.GetProduct(context.Background(), getReq)
	assert.NoError(t, err)
	assert.NotNil(t, getResp)
	assert.Equal(t, createReq.Name, getResp.Name)
	assert.Equal(t, createReq.Price, getResp.Price)
	assert.Equal(t, createReq.Description, getResp.Description)
	assert.Equal(t, category.ID, uint(getResp.Category.Id))
	assert.Equal(t, int32(30), getResp.Inventory.Stock)
}

// TestListProducts tests the ListProducts RPC method.
func TestListProducts(t *testing.T) {
	// Create a category and several products
	category := createTestCategory(t, "Home")
	for i := 1; i <= 3; i++ {
		req := &pb.CreateProductRequest{
			Name:        fmt.Sprintf("Home Product %d", i),
			Price:       float64(i) * 10.0,
			CategoryId:  uint32(category.ID),
			Description: fmt.Sprintf("Description for product %d", i),
			Stock:       int32(10 * i),
		}
		_, err := productServer.CreateProduct(context.Background(), req)
		assert.NoError(t, err)
	}
	listReq := &pb.ListProductsRequest{
		Page:  1,
		Limit: 10,
	}
	listResp, err := productServer.ListProducts(context.Background(), listReq)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)
	// Expect at least three products
	assert.GreaterOrEqual(t, len(listResp.Products), 3)
}

// TestUpdateProduct tests updating an existing product.
func TestUpdateProduct(t *testing.T) {
	category := createTestCategory(t, "Fashion")
	// Create a product
	createReq := &pb.CreateProductRequest{
		Name:        "Original Product",
		Price:       49.99,
		CategoryId:  uint32(category.ID),
		Description: "Original description",
		Stock:       15,
	}
	createResp, err := productServer.CreateProduct(context.Background(), createReq)
	assert.NoError(t, err)
	assert.NotNil(t, createResp)

	// Update the product
	updateReq := &pb.UpdateProductRequest{
		Id:          createResp.Id,
		Name:        "Updated Product",
		Price:       59.99,
		CategoryId:  uint32(category.ID),
		Description: "Updated description",
	}
	updateResp, err := productServer.UpdateProduct(context.Background(), updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updateResp)
	assert.Equal(t, updateReq.Name, updateResp.Name)
	assert.Equal(t, updateReq.Price, updateResp.Price)
	assert.Equal(t, updateReq.Description, updateResp.Description)
}

// TestDeleteProduct tests deleting a product.
func TestDeleteProduct(t *testing.T) {
	category := createTestCategory(t, "Toys")
	// Create a product
	createReq := &pb.CreateProductRequest{
		Name:        "Toy Product",
		Price:       29.99,
		CategoryId:  uint32(category.ID),
		Description: "A fun toy",
		Stock:       20,
	}
	createResp, err := productServer.CreateProduct(context.Background(), createReq)
	assert.NoError(t, err)
	assert.NotNil(t, createResp)

	// Delete the product
	delReq := &pb.DeleteProductRequest{Id: createResp.Id}
	delResp, err := productServer.DeleteProduct(context.Background(), delReq)
	assert.NoError(t, err)
	assert.True(t, delResp.Success)

	// Verify that the product is deleted by attempting to get it
	getReq := &pb.GetProductRequest{Id: createResp.Id}
	getResp, err := productServer.GetProduct(context.Background(), getReq)
	// We expect an error when trying to get a deleted product
	assert.Error(t, err)
	assert.Nil(t, getResp)
}

// TestSearchProducts tests the search functionality.
func TestSearchProducts(t *testing.T) {
	category := createTestCategory(t, "Sports")
	// Create a product with a unique name
	createReq := &pb.CreateProductRequest{
		Name:        "Soccer Ball",
		Price:       25.00,
		CategoryId:  uint32(category.ID),
		Description: "Standard size soccer ball",
		Stock:       100,
	}
	_, err := productServer.CreateProduct(context.Background(), createReq)
	assert.NoError(t, err)

	// Search by name (case-insensitive)
	searchReq := &pb.SearchRequest{
		Query: "soccer",
	}
	searchResp, err := productServer.SearchProducts(context.Background(), searchReq)
	assert.NoError(t, err)
	assert.NotNil(t, searchResp)
	// At least one product should be returned
	assert.GreaterOrEqual(t, len(searchResp.Products), 1)

	// Search by category (using lower-case for match)
	searchReq2 := &pb.SearchRequest{
		Query:    "soccer",
		Category: "sports",
	}
	searchResp2, err := productServer.SearchProducts(context.Background(), searchReq2)
	assert.NoError(t, err)
	assert.NotNil(t, searchResp2)
	assert.GreaterOrEqual(t, len(searchResp2.Products), 1)
}
