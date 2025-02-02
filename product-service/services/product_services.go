// product-service/services/product_service.go
package services

import (
	"context"
	"errors"
	"fmt"

	"product-service/db"
	"product-service/models"
	"product-service/proto"
)

type ProductServer struct {
	proto.UnimplementedProductServiceServer
}

func (s *ProductServer) CreateProduct(ctx context.Context, req *proto.CreateProductRequest) (*proto.ProductResponse, error) {
	tx := db.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create product
	product := models.Product{
		Name:        req.Name,
		Price:       req.Price,
		CategoryID:  uint(req.CategoryId),
		Description: req.Description,
	}

	if err := tx.Create(&product).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create product: %v", err)
	}

	// In the CreateProduct method, modify the inventory creation:
	inventory := models.Inventory{
		ProductID: product.ID,
		Stock:     int(req.Stock), // Convert int32 to int
	}

	if err := tx.Create(&inventory).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create inventory: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("transaction failed: %v", err)
	}

	return s.getProductResponse(product.ID)
}

func (s *ProductServer) GetProduct(ctx context.Context, req *proto.GetProductRequest) (*proto.ProductResponse, error) {
	return s.getProductResponse(uint(req.Id))
}

func (s *ProductServer) ListProducts(ctx context.Context, req *proto.ListProductsRequest) (*proto.ListProductsResponse, error) {
	var products []models.Product
	query := db.DB.Preload("Category").Preload("Inventory")

	if req.Limit > 0 {
		offset := (req.Page - 1) * req.Limit
		query = query.Offset(int(offset)).Limit(int(req.Limit))
	}

	if err := query.Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to list products: %v", err)
	}

	response := &proto.ListProductsResponse{}
	for _, p := range products {
		response.Products = append(response.Products, convertToProtoProduct(p))
	}

	return response, nil
}

func (s *ProductServer) UpdateProduct(ctx context.Context, req *proto.UpdateProductRequest) (*proto.ProductResponse, error) {
	var product models.Product
	if err := db.DB.First(&product, req.Id).Error; err != nil {
		return nil, fmt.Errorf("product not found: %v", err)
	}

	product.Name = req.Name
	product.Price = req.Price
	product.CategoryID = uint(req.CategoryId)
	product.Description = req.Description

	if err := db.DB.Save(&product).Error; err != nil {
		return nil, fmt.Errorf("failed to update product: %v", err)
	}

	return s.getProductResponse(uint(req.Id))
}

func (s *ProductServer) DeleteProduct(ctx context.Context, req *proto.DeleteProductRequest) (*proto.DeleteResponse, error) {
	result := db.DB.Delete(&models.Product{}, req.Id)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to delete product: %v", result.Error)
	}

	return &proto.DeleteResponse{Success: result.RowsAffected > 0}, nil
}

func (s *ProductServer) SearchProducts(ctx context.Context, req *proto.SearchRequest) (*proto.ListProductsResponse, error) {
	var products []models.Product
	query := db.DB.Preload("Category").Preload("Inventory").
		Where("LOWER(name) LIKE ?", fmt.Sprintf("%%%s%%", req.Query))

	if req.Category != "" {
		query = query.Joins("JOIN categories ON categories.id = products.category_id").
			Where("LOWER(categories.name) = ?", req.Category)
	}

	if err := query.Find(&products).Error; err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	response := &proto.ListProductsResponse{}
	for _, p := range products {
		response.Products = append(response.Products, convertToProtoProduct(p))
	}

	return response, nil
}

// Helper functions
func (s *ProductServer) getProductResponse(id uint) (*proto.ProductResponse, error) {
	var product models.Product
	if err := db.DB.Preload("Category").Preload("Inventory").First(&product, id).Error; err != nil {
		return nil, errors.New("product not found")
	}
	return convertToProtoProduct(product), nil
}

func convertToProtoProduct(p models.Product) *proto.ProductResponse {
	return &proto.ProductResponse{
		Id:          uint32(p.ID),
		Name:        p.Name,
		Price:       p.Price,
		Description: p.Description,
		Category: &proto.Category{
			Id:   uint32(p.Category.ID),
			Name: p.Category.Name,
		},
		Inventory: &proto.Inventory{
			ProductId: uint32(p.Inventory.ProductID),
			Stock:     int32(p.Inventory.Stock),
		},
	}
}
