package handlers

import (
	"context"
	"log"
	"net/http"

	pb "github.com/geoo115/E-commerceMicroservices/auth-service/proto"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Helper function to create a gRPC client connection
func getAuthServiceClient() (pb.AuthServiceClient, *grpc.ClientConn, error) {
	addr := viper.GetString("auth-service.address")
	if addr == "" {
		log.Fatal("Auth service address not set in config")
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := pb.NewAuthServiceClient(conn)
	return client, conn, nil
}

// Signup handler for creating a new user
func Signup(c *gin.Context) {
	var req pb.SignupRequest
	// Bind incoming JSON request to SignupRequest struct
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Signup request binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Log the incoming request for debugging purposes
	log.Printf("Received signup request: %+v", req)

	// Create gRPC client connection
	client, conn, err := getAuthServiceClient()
	if err != nil {
		log.Printf("Error connecting to Auth service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to connect to Auth service"})
		return
	}
	defer conn.Close()

	// Call the Signup method on the Auth service
	resp, err := client.Signup(context.Background(), &req)
	if err != nil {
		log.Printf("Signup error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Use only the fields available in GenericResponse
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": resp.Message, // Direct field access for GenericResponse
	})
}

// Login handler for user authentication
func Login(c *gin.Context) {
	var req pb.LoginRequest
	// Bind the incoming login request to LoginRequest struct
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Login request binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Log the incoming request for debugging purposes
	log.Printf("Received login request: %+v", req)

	// Create gRPC client connection
	client, conn, err := getAuthServiceClient()
	if err != nil {
		log.Printf("Error connecting to Auth service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to connect to Auth service"})
		return
	}
	defer conn.Close()

	// Call the Login method on the Auth service
	resp, err := client.Login(context.Background(), &req)
	if err != nil {
		log.Printf("Login error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Respond with the authentication token and user details
	c.JSON(http.StatusOK, gin.H{
		"user_id":      resp.GetUserId(),
		"username":     resp.GetUsername(),
		"email":        resp.GetEmail(),
		"access_token": resp.GetAccessToken(),
	})
}

// ValidateToken handler for validating the JWT token
func ValidateToken(c *gin.Context) {
	var req pb.ValidateTokenRequest
	// Bind incoming request to ValidateTokenRequest struct
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ValidateToken request binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Log the incoming request for debugging purposes
	log.Printf("Received validate token request: %+v", req)

	// Create gRPC client connection
	client, conn, err := getAuthServiceClient()
	if err != nil {
		log.Printf("Error connecting to Auth service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to connect to Auth service"})
		return
	}
	defer conn.Close()

	// Call the ValidateToken method on the Auth service
	resp, err := client.ValidateToken(context.Background(), &req)
	if err != nil {
		log.Printf("Token validation error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Respond with token validation result
	c.JSON(http.StatusOK, gin.H{
		"is_valid": resp.GetIsValid(),
		"user_id":  resp.GetUserId(),
		"role":     resp.GetRole(),
	})
}
