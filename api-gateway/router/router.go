package router

import (
	"github.com/geoo115/E-commerceMicroservices/api-gateway/handlers"
	"github.com/geoo115/E-commerceMicroservices/api-gateway/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Global middleware
	router.Use(middlewares.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Auth routes
	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/signup", handlers.Signup)
		authGroup.POST("/login", handlers.Login)
		authGroup.POST("/validate", handlers.ValidateToken)
	}

	// Product routes
	productGroup := router.Group("/api/v1/product")
	{
		productGroup.GET("/:id", handlers.GetProduct)
		productGroup.POST("/", handlers.CreateProduct)
		productGroup.PUT("/:id", handlers.UpdateProduct)
		productGroup.DELETE("/:id", handlers.DeleteProduct)
	}
	categoryGroup := router.Group("/api/v1/category")
	{
		categoryGroup.POST("/", handlers.CreateCategory)
	}

	// Order routes
	orderGroup := router.Group("/api/v1/order")
	{
		orderGroup.POST("/", handlers.CreateOrder)
		orderGroup.GET("/:id", handlers.GetOrder)
		orderGroup.GET("/user/:userId", handlers.GetUserOrders)
		orderGroup.PUT("/:id", handlers.UpdateOrderStatus)
	}

	// Cart routes
	cartGroup := router.Group("/api/v1/cart")
	{
		cartGroup.GET("/:userId", handlers.GetCart)
		cartGroup.POST("/add", handlers.AddToCart)
		cartGroup.POST("/remove", handlers.RemoveFromCart)
		// cartGroup.DELETE("/clear/:userId", handlers.ClearCart)
	}

	// Payment routes
	paymentGroup := router.Group("/api/v1/payment")
	{
		paymentGroup.POST("/process", handlers.ProcessPayment)
		paymentGroup.GET("/:orderId", handlers.GetPayment)
	}

	// Review routes
	reviewGroup := router.Group("/api/v1/review")
	{
		reviewGroup.POST("/", handlers.CreateReview)
		reviewGroup.GET("/:productId", handlers.ListProductReviews)
		reviewGroup.DELETE("/:reviewId", handlers.DeleteReview)
	}

	// Inventory routes
	inventoryGroup := router.Group("/api/v1/inventory")
	{
		inventoryGroup.GET("/:productId", handlers.GetInventory)
		inventoryGroup.POST("/update", handlers.UpdateInventory)
	}

	return router
}
