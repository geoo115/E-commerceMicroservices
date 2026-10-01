package handler

import (
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	mw "github.com/geoo115/E-commerceMicroservices/api-gateway/internal/middleware"
)

// NewRouter wires every route of the public API.
func NewRouter(h *Handler, log *slog.Logger) *gin.Engine {
	useJSONFieldNames()

	r := gin.New()
	r.Use(gin.Recovery(), mw.RequestID(), mw.Logger(log), mw.Metrics())
	r.HandleMethodNotAllowed = true

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	authn := mw.Authenticate(h.clients.Auth)
	admin := mw.RequireRole(mw.RoleAdmin)
	authLimit := mw.NewRateLimiter(20, 10).Middleware()

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth", authLimit)
	auth.POST("/signup", h.Signup)
	auth.POST("/verify-email", h.VerifyEmail)
	auth.POST("/resend-code", h.ResendVerificationCode)
	auth.POST("/login", h.Login)

	v1.GET("/users/me", authn, h.Me)

	v1.GET("/categories", h.ListCategories)
	v1.POST("/categories", authn, admin, h.CreateCategory)

	v1.GET("/products", h.ListProducts)
	v1.GET("/products/:id", h.GetProduct)
	v1.POST("/products", authn, admin, h.CreateProduct)
	v1.PUT("/products/:id", authn, admin, h.UpdateProduct)
	v1.DELETE("/products/:id", authn, admin, h.DeleteProduct)

	v1.GET("/products/:id/reviews", h.ListReviews)
	v1.POST("/products/:id/reviews", authn, h.CreateReview)
	v1.DELETE("/reviews/:id", authn, h.DeleteReview)

	inventory := v1.Group("/inventory", authn, admin)
	inventory.GET("/:productId", h.GetStock)
	inventory.POST("/:productId/adjustments", h.AdjustStock)

	wishlist := v1.Group("/wishlist", authn)
	wishlist.GET("", h.ListWishlist)
	wishlist.PUT("/:productId", h.AddToWishlist)
	wishlist.DELETE("/:productId", h.RemoveFromWishlist)

	cart := v1.Group("/cart", authn)
	cart.GET("", h.GetCart)
	cart.DELETE("", h.ClearCart)
	cart.POST("/items", h.AddCartItem)
	cart.PUT("/items/:productId", h.UpdateCartItem)
	cart.DELETE("/items/:productId", h.RemoveCartItem)
	cart.POST("/checkout", h.Checkout)

	orders := v1.Group("/orders", authn)
	orders.POST("", h.CreateOrder)
	orders.GET("", h.ListMyOrders)
	orders.GET("/:id", h.GetOrder)
	orders.POST("/:id/cancel", h.CancelOrder)
	orders.PATCH("/:id/status", admin, h.UpdateOrderStatus)
	orders.POST("/:id/payment", h.PayOrder)
	orders.GET("/:id/payment", h.GetOrderPayment)

	return r
}

// useJSONFieldNames makes validation errors report JSON field names
// ("price_cents") instead of Go field names ("PriceCents").
func useJSONFieldNames() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(f reflect.StructField) string {
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "-" {
				return ""
			}
			return name
		})
	}
}
