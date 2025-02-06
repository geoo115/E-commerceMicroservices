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

func getAuthServiceClient() (pb.AuthServiceClient, *grpc.ClientConn, error) {
	addr := viper.GetString("auth_service.address")
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

func Signup(c *gin.Context) {
	var req pb.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, conn, err := getAuthServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to auth service"})
		return
	}
	defer conn.Close()

	resp, err := client.Signup(context.Background(), &req)
	if err != nil {
		log.Printf("Signup error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func Login(c *gin.Context) {
	var req pb.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, conn, err := getAuthServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to auth service"})
		return
	}
	defer conn.Close()

	resp, err := client.Login(context.Background(), &req)
	if err != nil {
		log.Printf("Login error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func ValidateToken(c *gin.Context) {
	var req pb.ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, conn, err := getAuthServiceClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to connect to auth service"})
		return
	}
	defer conn.Close()

	resp, err := client.ValidateToken(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	c.JSON(http.StatusOK, resp)
}
