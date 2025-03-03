// db/database.go
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

func InitDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		GetEnv("DATABASE_HOST", "localhost"),
		GetEnv("DATABASE_USER", "usr"),
		GetEnv("DATABASE_PASSWORD", "test123"),
		GetEnv("DATABASE_NAME", "ecommerce_users"),
		GetEnv("DATABASE_PORT", "5432"),
		GetEnv("DATABASE_SSLMODE", "disable"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	DB = db

	// Migrate models
	if err := db.AutoMigrate(&models.User{}, &models.Address{}); err != nil {
		return nil, fmt.Errorf("database migration error: %w", err)
	}

	log.Println("✅ Database connected and migrated")
	return db, nil
}

func CloseDB() {
	if sqlDB, err := DB.DB(); err == nil {
		sqlDB.Close()
	} else {
		log.Fatal("Failed to close database connection")
	}
}

func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
