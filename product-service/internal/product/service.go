// Package product implements the product-service gRPC API (the catalog).
package product

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/geoo115/E-commerceMicroservices/pkg/cache"
	"github.com/geoo115/E-commerceMicroservices/pkg/events"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	"github.com/geoo115/E-commerceMicroservices/pkg/paging"
	productv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/product/v1"
)

const productCacheTTL = 5 * time.Minute

// Service implements productv1.ProductServiceServer.
type Service struct {
	productv1.UnimplementedProductServiceServer

	db    *gorm.DB
	cache cache.Cache
	log   *slog.Logger
}

// NewService returns a product Service.
func NewService(db *gorm.DB, c cache.Cache, log *slog.Logger) *Service {
	return &Service{db: db, cache: c, log: log}
}

func productKey(id uint64) string { return "product:" + strconv.FormatUint(id, 10) }

// CreateCategory adds a category.
func (s *Service) CreateCategory(ctx context.Context, req *productv1.CreateCategoryRequest) (*productv1.CreateCategoryResponse, error) {
	name := strings.TrimSpace(req.GetName())
	if name == "" || len(name) > 64 {
		return nil, status.Error(codes.InvalidArgument, "name must be 1-64 characters")
	}
	c := Category{Name: name}
	if err := s.db.WithContext(ctx).Create(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, status.Error(codes.AlreadyExists, "category already exists")
		}
		return nil, grpcx.Internal(ctx, s.log, "create category", err)
	}
	return &productv1.CreateCategoryResponse{Category: categoryToProto(c)}, nil
}

// ListCategories returns all categories.
func (s *Service) ListCategories(ctx context.Context, _ *productv1.ListCategoriesRequest) (*productv1.ListCategoriesResponse, error) {
	var cats []Category
	if err := s.db.WithContext(ctx).Order("name").Find(&cats).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "list categories", err)
	}
	out := make([]*productv1.Category, len(cats))
	for i, c := range cats {
		out[i] = categoryToProto(c)
	}
	return &productv1.ListCategoriesResponse{Categories: out}, nil
}

// CreateProduct adds a product and emits product.created so that
// inventory-service can initialise its stock.
func (s *Service) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductResponse, error) {
	if err := validateProduct(req.GetName(), req.GetPriceCents()); err != nil {
		return nil, err
	}
	if req.GetInitialStock() < 0 {
		return nil, status.Error(codes.InvalidArgument, "initial_stock must not be negative")
	}

	p := Product{
		Name:        strings.TrimSpace(req.GetName()),
		Description: req.GetDescription(),
		PriceCents:  req.GetPriceCents(),
		CategoryID:  req.GetCategoryId(),
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&p.Category, req.GetCategoryId()).Error; err != nil {
			return err
		}
		if err := tx.Create(&p).Error; err != nil {
			return err
		}
		ev, err := events.New(events.ProductCreated, events.ProductCreatedData{
			ProductID:    p.ID,
			InitialStock: req.GetInitialStock(),
		})
		if err != nil {
			return err
		}
		return events.Enqueue(tx, events.TopicProducts, strconv.FormatUint(p.ID, 10), ev)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.InvalidArgument, "category does not exist")
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "create product", err)
	}
	return &productv1.CreateProductResponse{Product: productToProto(p)}, nil
}

// GetProduct returns a product, served from Redis when cached (cache-aside).
func (s *Service) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.GetProductResponse, error) {
	key := productKey(req.GetId())
	if b, ok := s.cache.Get(ctx, key); ok {
		var cached productv1.Product
		if proto.Unmarshal(b, &cached) == nil {
			return &productv1.GetProductResponse{Product: &cached}, nil
		}
	}

	var p Product
	err := s.db.WithContext(ctx).Preload("Category").First(&p, req.GetId()).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "get product", err)
	}

	out := productToProto(p)
	if b, err := proto.Marshal(out); err == nil {
		s.cache.Set(ctx, key, b, productCacheTTL)
	}
	return &productv1.GetProductResponse{Product: out}, nil
}

// BatchGetProducts returns the requested products, failing if any is missing.
func (s *Service) BatchGetProducts(ctx context.Context, req *productv1.BatchGetProductsRequest) (*productv1.BatchGetProductsResponse, error) {
	ids := uniqueIDs(req.GetIds())
	if len(ids) == 0 {
		return &productv1.BatchGetProductsResponse{}, nil
	}
	if len(ids) > 100 {
		return nil, status.Error(codes.InvalidArgument, "at most 100 ids per request")
	}

	var products []Product
	if err := s.db.WithContext(ctx).Preload("Category").Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "batch get products", err)
	}
	if len(products) != len(ids) {
		found := make(map[uint64]bool, len(products))
		for _, p := range products {
			found[p.ID] = true
		}
		for _, id := range ids {
			if !found[id] {
				return nil, status.Errorf(codes.NotFound, "product %d not found", id)
			}
		}
	}

	out := make([]*productv1.Product, len(products))
	for i, p := range products {
		out[i] = productToProto(p)
	}
	return &productv1.BatchGetProductsResponse{Products: out}, nil
}

// ListProducts returns a page of products, optionally filtered by name and category.
func (s *Service) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	q := s.db.WithContext(ctx).Model(&Product{})
	if query := strings.TrimSpace(req.GetQuery()); query != "" {
		q = q.Where("name ILIKE ?", "%"+escapeLike(query)+"%")
	}
	if req.GetCategoryId() != 0 {
		q = q.Where("category_id = ?", req.GetCategoryId())
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "count products", err)
	}

	limit, offset := paging.Normalize(req.GetPage(), req.GetPageSize())
	var products []Product
	if err := q.Preload("Category").Order("id").Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "list products", err)
	}

	out := make([]*productv1.Product, len(products))
	for i, p := range products {
		out[i] = productToProto(p)
	}
	return &productv1.ListProductsResponse{Products: out, Total: total}, nil
}

// UpdateProduct replaces a product's editable fields.
func (s *Service) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.UpdateProductResponse, error) {
	if err := validateProduct(req.GetName(), req.GetPriceCents()); err != nil {
		return nil, err
	}

	var p Product
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&p, req.GetId()).Error; err != nil {
			return status.Error(codes.NotFound, "product not found")
		}
		if err := tx.First(&p.Category, req.GetCategoryId()).Error; err != nil {
			return status.Error(codes.InvalidArgument, "category does not exist")
		}
		p.Name = strings.TrimSpace(req.GetName())
		p.Description = req.GetDescription()
		p.PriceCents = req.GetPriceCents()
		p.CategoryID = req.GetCategoryId()
		return tx.Save(&p).Error
	})
	if _, ok := status.FromError(err); ok && err != nil {
		return nil, err
	}
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "update product", err)
	}

	s.cache.Delete(ctx, productKey(p.ID))
	return &productv1.UpdateProductResponse{Product: productToProto(p)}, nil
}

// DeleteProduct soft-deletes a product so existing orders keep their history.
func (s *Service) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*productv1.DeleteProductResponse, error) {
	res := s.db.WithContext(ctx).Delete(&Product{}, req.GetId())
	if res.Error != nil {
		return nil, grpcx.Internal(ctx, s.log, "delete product", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	s.cache.Delete(ctx, productKey(req.GetId()))
	return &productv1.DeleteProductResponse{}, nil
}

// HandleReviewEvent keeps the rating aggregates in sync with review-service.
func (s *Service) HandleReviewEvent(ctx context.Context, ev events.Event) error {
	var sign int64
	switch ev.Type {
	case events.ReviewCreated:
		sign = 1
	case events.ReviewDeleted:
		sign = -1
	default:
		return nil
	}

	var data events.ReviewData
	if err := ev.Decode(&data); err != nil {
		return err
	}
	if data.Rating < 1 || data.Rating > 5 {
		return fmt.Errorf("invalid rating %d", data.Rating)
	}

	err := events.ProcessOnce(ctx, s.db, "product-service.ratings", ev, func(tx *gorm.DB) error {
		return tx.Model(&Product{}).Unscoped().
			Where("id = ? AND rating_count + ? >= 0", data.ProductID, sign).
			Updates(map[string]any{
				"rating_count": gorm.Expr("rating_count + ?", sign),
				"rating_sum":   gorm.Expr("rating_sum + ?", sign*int64(data.Rating)),
			}).Error
	})
	if err != nil {
		return err
	}
	s.cache.Delete(ctx, productKey(data.ProductID))
	return nil
}

func validateProduct(name string, priceCents int64) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return status.Error(codes.InvalidArgument, "name must be 1-200 characters")
	}
	if priceCents <= 0 {
		return status.Error(codes.InvalidArgument, "price_cents must be positive")
	}
	return nil
}

func uniqueIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]bool, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string { return likeEscaper.Replace(s) }

func categoryToProto(c Category) *productv1.Category {
	return &productv1.Category{Id: c.ID, Name: c.Name}
}

func productToProto(p Product) *productv1.Product {
	return &productv1.Product{
		Id:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		PriceCents:    p.PriceCents,
		Category:      categoryToProto(p.Category),
		RatingAverage: p.RatingAverage(),
		RatingCount:   p.RatingCount,
	}
}
