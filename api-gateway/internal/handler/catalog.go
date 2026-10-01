package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/inventory/v1"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
)

type categoryRequest struct {
	Name string `json:"name" binding:"required,max=64"`
}

// CreateCategory adds a category (admin). POST /api/v1/categories
func (h *Handler) CreateCategory(c *gin.Context) {
	var req categoryRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.clients.Product.CreateCategory(c.Request.Context(), &productv1.CreateCategoryRequest{Name: req.Name})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusCreated, categoryDTO{ID: resp.GetCategory().GetId(), Name: resp.GetCategory().GetName()})
}

// ListCategories returns all categories. GET /api/v1/categories
func (h *Handler) ListCategories(c *gin.Context) {
	resp, err := h.clients.Product.ListCategories(c.Request.Context(), &productv1.ListCategoriesRequest{})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": mapSlice(resp.GetCategories(), func(cat *productv1.Category) categoryDTO {
		return categoryDTO{ID: cat.GetId(), Name: cat.GetName()}
	})})
}

type productRequest struct {
	Name         string `json:"name" binding:"required,max=200"`
	Description  string `json:"description" binding:"max=5000"`
	PriceCents   int64  `json:"price_cents" binding:"required,gt=0"`
	CategoryID   uint64 `json:"category_id" binding:"required"`
	InitialStock int32  `json:"initial_stock" binding:"gte=0"`
}

// CreateProduct adds a product (admin). POST /api/v1/products
func (h *Handler) CreateProduct(c *gin.Context) {
	var req productRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.clients.Product.CreateProduct(c.Request.Context(), &productv1.CreateProductRequest{
		Name:         req.Name,
		Description:  req.Description,
		PriceCents:   req.PriceCents,
		CategoryId:   req.CategoryID,
		InitialStock: req.InitialStock,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProduct(resp.GetProduct()))
}

// GetProduct returns a product with its available stock. GET /api/v1/products/:id
//
// This is API composition: the price comes from product-service and the stock
// from inventory-service. If inventory is down the product is still served.
func (h *Handler) GetProduct(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.clients.Product.GetProduct(c.Request.Context(), &productv1.GetProductRequest{Id: id})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	out := toProduct(resp.GetProduct())

	ctx, cancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
	defer cancel()
	if stock, err := h.clients.Inventory.GetStock(ctx, &inventoryv1.GetStockRequest{ProductId: id}); err == nil {
		available := stock.GetAvailable()
		out.Available = &available
	} else if status.Code(err) != codes.NotFound { // NotFound: stock not initialised yet
		h.log.WarnContext(c.Request.Context(), "stock unavailable for product", "product_id", id, "error", err)
	}
	c.JSON(http.StatusOK, out)
}

// ListProducts returns a page of products. GET /api/v1/products?q=&category_id=&page=&page_size=
func (h *Handler) ListProducts(c *gin.Context) {
	pageNum, size := pageParams(c)
	categoryID, _ := strconv.ParseUint(c.Query("category_id"), 10, 64)
	resp, err := h.clients.Product.ListProducts(c.Request.Context(), &productv1.ListProductsRequest{
		Page:       pageNum,
		PageSize:   size,
		Query:      c.Query("q"),
		CategoryId: categoryID,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, page[productDTO]{
		Items:    mapSlice(resp.GetProducts(), toProduct),
		Total:    resp.GetTotal(),
		Page:     pageNum,
		PageSize: size,
	})
}

// UpdateProduct replaces a product's fields (admin). PUT /api/v1/products/:id
func (h *Handler) UpdateProduct(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var req productRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.clients.Product.UpdateProduct(c.Request.Context(), &productv1.UpdateProductRequest{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		CategoryId:  req.CategoryID,
	})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProduct(resp.GetProduct()))
}

// DeleteProduct removes a product (admin). DELETE /api/v1/products/:id
func (h *Handler) DeleteProduct(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if _, err := h.clients.Product.DeleteProduct(c.Request.Context(), &productv1.DeleteProductRequest{Id: id}); err != nil {
		h.grpcError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetStock returns the stock of a product (admin). GET /api/v1/inventory/:productId
func (h *Handler) GetStock(c *gin.Context) {
	id, ok := pathID(c, "productId")
	if !ok {
		return
	}
	resp, err := h.clients.Inventory.GetStock(c.Request.Context(), &inventoryv1.GetStockRequest{ProductId: id})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toStock(resp))
}

type adjustStockRequest struct {
	Delta int32 `json:"delta" binding:"required"`
}

// AdjustStock restocks or writes off units (admin). POST /api/v1/inventory/:productId/adjustments
func (h *Handler) AdjustStock(c *gin.Context) {
	id, ok := pathID(c, "productId")
	if !ok {
		return
	}
	var req adjustStockRequest
	if !bindJSON(c, &req) {
		return
	}
	resp, err := h.clients.Inventory.AdjustStock(c.Request.Context(), &inventoryv1.AdjustStockRequest{ProductId: id, Delta: req.Delta})
	if err != nil {
		h.grpcError(c, err)
		return
	}
	c.JSON(http.StatusOK, toStock(resp))
}

// ensureProductExists writes 404 and returns false if the product does not exist.
func (h *Handler) ensureProductExists(c *gin.Context, id uint64) bool {
	if _, err := h.clients.Product.GetProduct(c.Request.Context(), &productv1.GetProductRequest{Id: id}); err != nil {
		h.grpcError(c, err)
		return false
	}
	return true
}
