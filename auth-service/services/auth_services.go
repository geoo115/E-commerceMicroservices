package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/geoo115/E-commerceMicroservices/auth-service/cache"
	"github.com/geoo115/E-commerceMicroservices/auth-service/db"
	"github.com/geoo115/E-commerceMicroservices/auth-service/models"
	"github.com/geoo115/E-commerceMicroservices/auth-service/proto"
	"github.com/geoo115/E-commerceMicroservices/auth-service/utils"

	"gorm.io/gorm"
)

// AuthServer implements the gRPC AuthService interface.
type AuthServer struct {
	proto.UnimplementedAuthServiceServer
}

// NewAuthServer returns a new AuthServer instance.
func NewAuthServer() *AuthServer {
	return &AuthServer{}
}

// Signup registers a new user and creates an associated address record if provided.
func (s *AuthServer) Signup(ctx context.Context, req *proto.SignupRequest) (*proto.AuthResponse, error) {
	// Validate required fields.
	if req.Username == "" || req.Password == "" || req.Email == "" || req.Phone == "" {
		return nil, fmt.Errorf("username, password, email, and phone are required")
	}

	// Check if the user already exists.
	var existing models.User
	result := db.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existing)
	if result.Error == nil {
		return nil, fmt.Errorf("username or email already exists")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Printf("Database error: %v", result.Error)
		return nil, fmt.Errorf("internal server error")
	}

	// Hash the password.
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	// Create the new user.
	user := models.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		Phone:    req.Phone,
		Role:     "customer", // Default role.
	}

	if err := db.DB.Create(&user).Error; err != nil {
		log.Printf("User creation error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	// If address details are provided, create an address record.
	if req.Address != "" && req.City != "" && req.PostCode != "" {
		address := models.Address{
			UserID:   user.ID,
			Address:  req.Address,
			City:     req.City,
			PostCode: req.PostCode,
		}
		if err := db.DB.Create(&address).Error; err != nil {
			log.Printf("Address creation error: %v", err)
			// You can decide whether to treat this as fatal or simply log the error.
		}
	}

	// Generate a JWT token.
	token, err := utils.GenerateToken(user)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	// Cache the token in Redis.
	cache.RedisClient.Set(ctx, fmt.Sprintf("auth_token:%s", token), user.ID, 24*time.Hour)

	return &proto.AuthResponse{
		UserId:      uint64(user.ID),
		AccessToken: token,
		Username:    user.Username,
		Email:       user.Email,
	}, nil
}

// Login authenticates a user and returns a JWT token.
func (s *AuthServer) Login(ctx context.Context, req *proto.LoginRequest) (*proto.AuthResponse, error) {
	var user models.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid credentials")
		}
		log.Printf("Database error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	token, err := utils.GenerateToken(user)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		return nil, fmt.Errorf("internal server error")
	}

	cache.RedisClient.Set(ctx, fmt.Sprintf("auth_token:%s", token), user.ID, 24*time.Hour)

	return &proto.AuthResponse{
		UserId:      uint64(user.ID),
		AccessToken: token,
		Username:    user.Username,
		Email:       user.Email,
	}, nil
}

// ValidateToken checks if a JWT token is valid using Redis caching.
func (s *AuthServer) ValidateToken(ctx context.Context, req *proto.ValidateTokenRequest) (*proto.ValidateResponse, error) {
	// Check Redis first.
	cached, err := cache.RedisClient.Get(ctx, fmt.Sprintf("auth_token:%s", req.Token)).Result()
	if err == nil && cached != "" {
		var userID uint64
		_, err := fmt.Sscanf(cached, "%d", &userID)
		if err == nil {
			// For simplicity, we assume role "customer" here.
			return &proto.ValidateResponse{
				IsValid: true,
				UserId:  userID,
				Role:    "customer",
			}, nil
		}
	}

	// Otherwise, validate the token using JWT.
	claims, err := utils.ValidateToken(req.Token)
	if err != nil {
		return &proto.ValidateResponse{IsValid: false}, nil
	}

	// Cache the result.
	if _, err := json.Marshal(claims); err == nil {
		cache.RedisClient.Set(ctx, fmt.Sprintf("auth_token:%s", req.Token), claims.UserID, 24*time.Hour)
	}

	return &proto.ValidateResponse{
		IsValid: true,
		UserId:  uint64(claims.UserID),
		Role:    claims.Role,
	}, nil
}
