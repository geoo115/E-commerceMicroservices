package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/geoo115/E-commerceMicroservices/auth-service/cache"
	"github.com/geoo115/E-commerceMicroservices/auth-service/db"
	"github.com/geoo115/E-commerceMicroservices/auth-service/models"
	"github.com/geoo115/E-commerceMicroservices/auth-service/proto"
	"github.com/geoo115/E-commerceMicroservices/auth-service/services"
	"github.com/geoo115/E-commerceMicroservices/auth-service/utils"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var authServer *services.AuthServer

func TestMain(m *testing.M) {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		log.Fatal("Error loading .env file")
		os.Exit(1)
	}

	// Initialize the database connection
	_, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize the database: %v", err)
		os.Exit(1)
	}
	log.Println("Database connected and migrated")

	// Initialize Redis
	cache.InitRedis() // Redis client is initialized here
	log.Println("Connected to Redis")

	// Initialize the auth server
	authServer = services.NewAuthServer()
	os.Exit(m.Run())
}

func TestSignup(t *testing.T) {
	req := &proto.SignupRequest{
		Username: "testuser5",
		Password: "password123",
		Email:    "test5@example.com",
		Phone:    "12322678099",
	}

	resp, err := authServer.Signup(context.Background(), req)

	// Assert no error and the response is as expected
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Signup successful. Please verify your email", resp.Message)

	// Verify that the user is created in the database and inactive
	var user models.User
	err = db.DB.Where("username = ?", req.Username).First(&user).Error
	assert.Nil(t, err)
	assert.Equal(t, req.Username, user.Username)
	assert.Equal(t, "customer", user.Role)
	assert.False(t, user.IsActive) // Ensure the user is not active yet

	// Verify that the Redis key was created for email verification
	verificationCode, err := cache.RedisClient.Get(context.Background(), fmt.Sprintf("verify:%s", user.Email)).Result()
	assert.Nil(t, err)
	assert.NotNil(t, verificationCode) // Verify that a verification code exists
}

func TestLogin(t *testing.T) {
	// Assuming the test user has already signed up and verified their email
	req := &proto.LoginRequest{
		Username: "testuser1",
		Password: "password123",
	}

	resp, err := authServer.Login(context.Background(), req)

	// Assert no error and the response is as expected
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "testuser1", resp.Username)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestLoginInactiveUser(t *testing.T) {
	// Try to login with an inactive user
	req := &proto.LoginRequest{
		Username: "testuser5", // This user is inactive after signup
		Password: "password123",
	}

	resp, err := authServer.Login(context.Background(), req)

	// Assert that the error is permission denied due to email not being verified
	assert.NotNil(t, err)
	assert.Equal(t, "Verify your email first", err.Error())
	assert.Nil(t, resp)
}

func TestValidateToken(t *testing.T) {
	// Create a test user and generate a token
	user := models.User{
		Model:    gorm.Model{ID: 1},
		Username: "testuser1",
		Email:    "test1@example.com",
		Role:     "customer",
	}

	token, err := utils.GenerateToken(user)
	if err != nil {
		t.Fatalf("Token generation failed: %v", err)
	}

	req := &proto.ValidateTokenRequest{Token: token}

	resp, err := authServer.ValidateToken(context.Background(), req)

	// Assert no error and the token is valid
	assert.Nil(t, err)
	assert.True(t, resp.IsValid)
	assert.Equal(t, uint64(1), resp.UserId)
	assert.Equal(t, "customer", resp.Role)
}

func TestValidateInvalidToken(t *testing.T) {
	// Use an invalid token (this is a mock scenario)
	invalidToken := "invalid.token.value"
	req := &proto.ValidateTokenRequest{Token: invalidToken}

	resp, err := authServer.ValidateToken(context.Background(), req)

	// Assert that there is an error due to invalid token
	assert.NotNil(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "token is invalid")
}
