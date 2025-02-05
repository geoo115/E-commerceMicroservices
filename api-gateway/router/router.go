package router

import (
	"api-gateway/handlers"
	"api-gateway/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Global middleware
	router.Use(middlewares.Logger())

	// Auth routes
	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/signup", handlers.Signup)
		authGroup.POST("/login", handlers.Login)
	}

	// Product routes
	productGroup := router.Group("/api/v1/product")
	{
		productGroup.GET("/:id", handlers.GetProduct)
		productGroup.POST("/", handlers.CreateProduct)
		// You can add more endpoints as needed.
	}

	// Add other microservice routes (order, cart, payment, review, inventory) as needed.

	return router
}
