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

// getAuthServiceClient creates a gRPC client connection to the Auth service.
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

// Signup handles the user signup request.
func Signup(c *gin.Context) {
	var req pb.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Signup request binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	log.Printf("Signup Request received: Username=%s, Email=%s",
		req.GetUsername(), req.GetEmail())

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

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"user_id":  resp.GetUserId(),
		"username": resp.GetUsername(),
		"email":    resp.GetEmail(),
		"message":  "Signup successful. Please verify your email",
	})
}

// Login handles user login.
func Login(c *gin.Context) {
	var req pb.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Login request binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	log.Printf("Received login request: Username=%s", req.Username)

	client, conn, err := getAuthServiceClient()
	if err != nil {
		log.Printf("Error connecting to Auth service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to connect to Auth service"})
		return
	}
	defer conn.Close()

	resp, err := client.Login(context.Background(), &req)
	if err != nil {
		log.Printf("Login error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":      resp.GetUserId(),
		"username":     resp.GetUsername(),
		"email":        resp.GetEmail(),
		"access_token": resp.GetAccessToken(),
	})
}

// ValidateToken handles token validation.
func ValidateToken(c *gin.Context) {
	var req pb.ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ValidateToken request binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	log.Printf("Received validate token request: Token=%s", req.Token)

	client, conn, err := getAuthServiceClient()
	if err != nil {
		log.Printf("Error connecting to Auth service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to connect to Auth service"})
		return
	}
	defer conn.Close()

	resp, err := client.ValidateToken(context.Background(), &req)
	if err != nil {
		log.Printf("Token validation error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_valid": resp.GetIsValid(),
		"user_id":  resp.GetUserId(),
		"role":     resp.GetRole(),
	})
}
