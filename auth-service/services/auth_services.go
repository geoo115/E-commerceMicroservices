// services/auth_service.go
package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/geoo115/E-commerceMicroservices/auth-service/cache"
	"github.com/geoo115/E-commerceMicroservices/auth-service/db"
	"github.com/geoo115/E-commerceMicroservices/auth-service/models"
	"github.com/geoo115/E-commerceMicroservices/auth-service/proto"
	"github.com/geoo115/E-commerceMicroservices/auth-service/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type AuthServer struct {
	proto.UnimplementedAuthServiceServer
}

func NewAuthServer() *AuthServer {
	return &AuthServer{}
}

func (s *AuthServer) Signup(ctx context.Context, req *proto.SignupRequest) (*proto.GenericResponse, error) {
	var existing models.User
	if err := db.DB.Where("username = ? OR email = ? OR phone = ?", req.GetUsername(), req.GetEmail(), req.GetPhone()).First(&existing).Error; err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "User already exists")
	} else if err != gorm.ErrRecordNotFound {
		log.Printf("Error checking user existence: %v", err)
		return nil, status.Errorf(codes.Internal, "Error checking user existence")
	}

	hashedPassword, err := utils.HashPassword(req.GetPassword())
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to hash password")
	}

	user := models.User{
		Username: req.GetUsername(),
		Password: hashedPassword,
		Email:    req.GetEmail(),
		Phone:    req.GetPhone(),
		Role:     "customer",
		IsActive: false,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		log.Printf("Error creating user: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to create user")
	}

	log.Printf("Signup Request received: Username=%s, Email=%s, Address=%s, City=%s, PostCode=%s",
		req.GetUsername(), req.GetEmail(), req.GetAddress(), req.GetCity(), req.GetPostCode())

	if req.GetAddress() != "" {
		address := models.Address{
			UserID:   user.ID,
			Address:  req.GetAddress(),
			City:     req.GetCity(),
			PostCode: req.GetPostCode(),
		}

		if err := db.DB.Create(&address).Error; err != nil {
			log.Printf("Error creating address for user %d: %v", user.ID, err)
			// You might want to return the error instead of continuing
		} else {
			log.Printf("Address created successfully for user %d: %s, %s, %s",
				user.ID, address.Address, address.City, address.PostCode)
		}
	}

	// Send email verification
	verificationCode := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	err = cache.RedisClient.Set(ctx, fmt.Sprintf("verify:%s", user.Email), verificationCode, 10*time.Minute).Err()
	if err != nil {
		log.Printf("Error saving verification code to Redis: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to save verification code")
	}

	// TODO: Send actual email with verification code
	log.Printf("Verification code for %s: %s", user.Email, verificationCode)

	return &proto.GenericResponse{Message: "Signup successful. Please verify your email"}, nil
}

// Verify Email
func (s *AuthServer) VerifyEmail(ctx context.Context, req *proto.VerifyEmailRequest) (*proto.GenericResponse, error) {
	code, err := cache.RedisClient.Get(ctx, fmt.Sprintf("verify:%s", req.GetEmail())).Result()
	if err != nil {
		log.Printf("Error retrieving verification code: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid or expired verification code")
	}

	if code != req.GetCode() {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid verification code")
	}

	result := db.DB.Model(&models.User{}).Where("email = ?", req.GetEmail()).Update("is_active", true)
	if result.Error != nil {
		log.Printf("Error updating user: %v", result.Error)
		return nil, status.Errorf(codes.Internal, "Failed to verify email")
	}

	if result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "User not found")
	}

	// Delete the verification code from Redis
	cache.RedisClient.Del(ctx, fmt.Sprintf("verify:%s", req.GetEmail()))

	return &proto.GenericResponse{Message: "Email verified successfully"}, nil
}

// Login
func (s *AuthServer) Login(ctx context.Context, req *proto.LoginRequest) (*proto.AuthResponse, error) {
	var user models.User
	if err := db.DB.Where("username = ?", req.GetUsername()).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "User not found")
		}
		log.Printf("Database error during login: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to retrieve user")
	}

	if !user.IsActive {
		return nil, status.Errorf(codes.PermissionDenied, "Email verification required")
	}

	if !utils.CheckPasswordHash(req.GetPassword(), user.Password) {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid credentials")
	}

	token, err := utils.GenerateToken(user)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to generate token")
	}

	return &proto.AuthResponse{
		UserId:      uint64(user.ID),
		AccessToken: token,
		Username:    user.Username,
		Email:       user.Email,
	}, nil
}

// ValidateToken implements the token validation endpoint
func (s *AuthServer) ValidateToken(ctx context.Context, req *proto.ValidateTokenRequest) (*proto.ValidateResponse, error) {
	claims, err := utils.ValidateToken(req.GetToken())
	if err != nil {
		log.Printf("Token validation error: %v", err)
		return &proto.ValidateResponse{IsValid: false}, nil
	}

	return &proto.ValidateResponse{
		IsValid: true,
		UserId:  uint64(claims.UserID),
		Role:    claims.Role,
	}, nil
}

func CreateAdminIfNotExists() {
	var admin models.User
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	adminPhone := os.Getenv("ADMIN_PHONE")

	if adminEmail == "" || adminPassword == "" {
		log.Println("⚠️ ADMIN_EMAIL or ADMIN_PASSWORD not set. Skipping admin creation.")
		return
	}

	if adminPhone == "" {
		adminPhone = "00000000" // Default phone number if not provided
	}

	// Check if an admin user already exists
	result := db.DB.Where("email = ? AND role = ?", adminEmail, "admin").First(&admin)
	if result.Error == nil {
		log.Println("✅ Admin user already exists. Skipping creation.")
		return
	}

	// Create new admin user
	hashedPassword, err := utils.HashPassword(adminPassword)
	if err != nil {
		log.Fatalf("❌ Failed to hash admin password: %v", err)
	}

	newAdmin := models.User{
		Username: "admin",
		Email:    adminEmail,
		Phone:    adminPhone,
		Password: hashedPassword,
		Role:     "admin",
		IsActive: true, // Admin is always active
	}

	if err := db.DB.Create(&newAdmin).Error; err != nil {
		log.Fatalf("❌ Failed to create admin user: %v", err)
	} else {
		log.Println("✅ Admin user created successfully")
	}
}
