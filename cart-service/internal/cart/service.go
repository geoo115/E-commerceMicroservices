// Package cart implements cart-service: carts stored in PostgreSQL and read
// through a Redis cache (cache-aside with invalidation on every write).
package cart

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/geoo115/E-commerceMicroservices/pkg/cache"
	"github.com/geoo115/E-commerceMicroservices/pkg/grpcx"
	cartv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/cart/v1"
)

const (
	maxQuantity  = 100
	cartCacheTTL = 10 * time.Minute
)

// Item is one product in a user's cart.
type Item struct {
	UserID    uint64 `gorm:"primaryKey;autoIncrement:false"`
	ProductID uint64 `gorm:"primaryKey;autoIncrement:false"`
	Quantity  int32  `gorm:"not null;check:quantity > 0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName implements gorm's Tabler.
func (Item) TableName() string { return "cart_items" }

// Models lists the tables owned by cart-service.
var Models = []any{&Item{}}

// Service implements cartv1.CartServiceServer.
type Service struct {
	cartv1.UnimplementedCartServiceServer

	db    *gorm.DB
	cache cache.Cache
	log   *slog.Logger
}

// NewService returns a cart Service.
func NewService(db *gorm.DB, c cache.Cache, log *slog.Logger) *Service {
	return &Service{db: db, cache: c, log: log}
}

func cartKey(userID uint64) string { return "cart:" + strconv.FormatUint(userID, 10) }

// GetCart returns the user's cart.
func (s *Service) GetCart(ctx context.Context, req *cartv1.GetCartRequest) (*cartv1.GetCartResponse, error) {
	if b, ok := s.cache.Get(ctx, cartKey(req.GetUserId())); ok {
		var cached cartv1.Cart
		if proto.Unmarshal(b, &cached) == nil {
			return &cartv1.GetCartResponse{Cart: &cached}, nil
		}
	}
	c, err := s.load(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &cartv1.GetCartResponse{Cart: c}, nil
}

// AddItem adds quantity units of a product, merging with an existing line.
func (s *Service) AddItem(ctx context.Context, req *cartv1.AddItemRequest) (*cartv1.AddItemResponse, error) {
	if err := validate(req.GetUserId(), req.GetProductId(), req.GetQuantity(), 1); err != nil {
		return nil, err
	}
	item := Item{UserID: req.GetUserId(), ProductID: req.GetProductId(), Quantity: req.GetQuantity()}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "product_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"quantity":   gorm.Expr("LEAST(cart_items.quantity + excluded.quantity, ?)", maxQuantity),
			"updated_at": time.Now(),
		}),
	}).Create(&item).Error
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "add cart item", err)
	}
	c, err := s.invalidateAndLoad(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &cartv1.AddItemResponse{Cart: c}, nil
}

// UpdateItem sets the quantity of a product already in the cart; 0 removes it.
func (s *Service) UpdateItem(ctx context.Context, req *cartv1.UpdateItemRequest) (*cartv1.UpdateItemResponse, error) {
	if err := validate(req.GetUserId(), req.GetProductId(), req.GetQuantity(), 0); err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Where("user_id = ? AND product_id = ?", req.GetUserId(), req.GetProductId())
	var res *gorm.DB
	if req.GetQuantity() == 0 {
		res = q.Delete(&Item{})
	} else {
		res = q.Model(&Item{}).Update("quantity", req.GetQuantity())
	}
	if res.Error != nil {
		return nil, grpcx.Internal(ctx, s.log, "update cart item", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "product is not in the cart")
	}
	c, err := s.invalidateAndLoad(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &cartv1.UpdateItemResponse{Cart: c}, nil
}

// RemoveItem removes a product from the cart. Removing a missing product is a no-op.
func (s *Service) RemoveItem(ctx context.Context, req *cartv1.RemoveItemRequest) (*cartv1.RemoveItemResponse, error) {
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND product_id = ?", req.GetUserId(), req.GetProductId()).
		Delete(&Item{}).Error
	if err != nil {
		return nil, grpcx.Internal(ctx, s.log, "remove cart item", err)
	}
	c, err := s.invalidateAndLoad(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &cartv1.RemoveItemResponse{Cart: c}, nil
}

// ClearCart empties the cart.
func (s *Service) ClearCart(ctx context.Context, req *cartv1.ClearCartRequest) (*cartv1.ClearCartResponse, error) {
	if err := s.db.WithContext(ctx).Where("user_id = ?", req.GetUserId()).Delete(&Item{}).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "clear cart", err)
	}
	s.cache.Delete(ctx, cartKey(req.GetUserId()))
	return &cartv1.ClearCartResponse{}, nil
}

func (s *Service) invalidateAndLoad(ctx context.Context, userID uint64) (*cartv1.Cart, error) {
	s.cache.Delete(ctx, cartKey(userID))
	return s.load(ctx, userID)
}

// load reads the cart from PostgreSQL and refreshes the cache.
func (s *Service) load(ctx context.Context, userID uint64) (*cartv1.Cart, error) {
	var items []Item
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at").Find(&items).Error; err != nil {
		return nil, grpcx.Internal(ctx, s.log, "load cart", err)
	}
	c := &cartv1.Cart{UserId: userID}
	for _, it := range items {
		c.Items = append(c.Items, &cartv1.CartItem{ProductId: it.ProductID, Quantity: it.Quantity})
	}
	if b, err := proto.Marshal(c); err == nil {
		s.cache.Set(ctx, cartKey(userID), b, cartCacheTTL)
	}
	return c, nil
}

func validate(userID, productID uint64, quantity, minQuantity int32) error {
	if userID == 0 || productID == 0 {
		return status.Error(codes.InvalidArgument, "user_id and product_id are required")
	}
	if quantity < minQuantity || quantity > maxQuantity {
		return status.Errorf(codes.InvalidArgument, "quantity must be between %d and %d", minQuantity, maxQuantity)
	}
	return nil
}
