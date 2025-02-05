package db

import (
	"fmt"
	"log"
	"os"

	"github.com/geoo115/E-commerceMicroservices/auth-service/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB() (*gorm.DB, error) {
	// Ensure all required environment variables are set
	requiredVars := []string{"DATABASE_HOST", "DATABASE_USER", "DATABASE_PASSWORD", "DATABASE_NAME", "DATABASE_PORT", "DATABASE_SSLMODE"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatalf("Missing required environment variable: %s", v)
		}
	}

	// Use DATABASE_SSLMODE from environment variables
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASSWORD"),
		os.Getenv("DATABASE_NAME"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_SSLMODE"), // Dynamically include sslmode
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return nil, err
	}

	DB = db

	// Run migrations
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Database migration error: %v", err)
		return nil, err
	}

	log.Println("✅ Database connected successfully")
	return db, nil
}

func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get database instance")
	}
	sqlDB.Close()
}
