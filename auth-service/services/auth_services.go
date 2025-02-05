package services

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/geoo115/E-commerceMicroservices/auth-service/db"
	"github.com/geoo115/E-commerceMicroservices/auth-service/models"
	"github.com/geoo115/E-commerceMicroservices/auth-service/proto"
	"github.com/geoo115/E-commerceMicroservices/auth-service/utils"

	"gorm.io/gorm"
)

// AuthServer implements the gRPC AuthService interface
type AuthServer struct {
	proto.UnimplementedAuthServiceServer
}

// NewAuthServer initializes and returns an AuthServer instance
func NewAuthServer() *AuthServer {
	return &AuthServer{}
}

// Signup registers a new user
func (s *AuthServer) Signup(ctx context.Context, req *proto.SignupRequest) (*proto.AuthResponse, error) {
	// Validate input fields
	if req.Username == "" || req.Password == "" || req.Email == "" {
		return nil, fmt.Errorf("all fields are required")
	}

	// Check if user already exists (by username or email)
	var existing models.User
	result := db.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existing)
	if result.Error == nil {
		return nil, fmt.Errorf("username or email already exists")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Printf("Database error: %v", result.Error)
		return nil, fmt.Errorf("internal server error")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	// Create new user
	user := models.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		Phone:    req.Phone,
		Role:     "customer", // Default role, can be configured dynamically
	}

	if err := db.DB.Create(&user).Error; err != nil {
		log.Printf("User creation error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	return &proto.AuthResponse{
		UserId:      uint64(user.ID),
		AccessToken: token,
		Username:    user.Username,
		Email:       user.Email,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *AuthServer) Login(ctx context.Context, req *proto.LoginRequest) (*proto.AuthResponse, error) {
	// Find user by username
	var user models.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid credentials")
		}
		log.Printf("Database error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	// Check if password matches
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	return &proto.AuthResponse{
		UserId:      uint64(user.ID),
		AccessToken: token,
		Username:    user.Username,
		Email:       user.Email, // Fixed missing email field
	}, nil
}

// ValidateToken checks if a JWT token is valid
func (s *AuthServer) ValidateToken(ctx context.Context, req *proto.ValidateTokenRequest) (*proto.ValidateResponse, error) {
	claims, err := utils.ValidateToken(req.Token)
	if err != nil {
		return &proto.ValidateResponse{IsValid: false}, nil
	}

	return &proto.ValidateResponse{
		IsValid: true,
		UserId:  uint64(claims.UserID),
		Role:    claims.Role,
	}, nil
}
