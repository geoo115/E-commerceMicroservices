package tests

import (
	"auth-service/db"
	"auth-service/models"
	"auth-service/proto"
	"auth-service/services"
	"auth-service/utils"
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var authServer *services.AuthServer

func TestMain(m *testing.M) {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		os.Exit(1)
	}

	// Initialize database
	_, err := db.InitDB()
	if err != nil {
		os.Exit(1) // Fail if DB cannot be initialized
	}

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

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "testuser5", resp.Username)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestLogin(t *testing.T) {
	req := &proto.LoginRequest{
		Username: "testuser1",
		Password: "password123",
	}

	resp, err := authServer.Login(context.Background(), req)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "testuser1", resp.Username)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestValidateToken(t *testing.T) {
	user := models.User{
		Model:    gorm.Model{ID: 1},
		Username: "testuser1",
		Email:    "test1@example.com",
		Role:     "customer",
	}

	token, _ := utils.GenerateToken(user)

	req := &proto.ValidateTokenRequest{Token: token}

	resp, err := authServer.ValidateToken(context.Background(), req)

	assert.Nil(t, err)
	assert.True(t, resp.IsValid)
	assert.Equal(t, uint64(1), resp.UserId)
	assert.Equal(t, "customer", resp.Role)
}
