package db

import (
	"fmt"
	"log"
	"os"

	"github.com/geoo115/E-commerceMicroservices/inventory-service/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection and auto-migrates models.
func InitDB() {
	// Check required environment variables.
	requiredEnv := []string{"DATABASE_HOST", "DATABASE_USER", "DATABASE_PASSWORD", "DATABASE_NAME", "DATABASE_PORT", "DATABASE_SSLMODE"}
	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			log.Fatalf("Missing required environment variable: %s", env)
		}
	}

	// Create DSN.
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASSWORD"),
		os.Getenv("DATABASE_NAME"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_SSLMODE"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate models.
	if err = DB.AutoMigrate(&models.Inventory{}, &models.Product{}, &models.Category{}, &models.Order{}, &models.OrderItem{}); err != nil {
		log.Fatalf("Failed to auto-migrate models: %v", err)
	}

	log.Println("✅ Inventory Service Database connected successfully")
}

// CloseDB closes the database connection.
func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get database instance")
	}
	if err := sqlDB.Close(); err != nil {
		log.Fatalf("Failed to close database connection: %v", err)
	}
	log.Println("✅ Inventory Service Database connection closed")
}
