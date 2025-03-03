// main.go
package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/auth-service/cache"
	"github.com/geoo115/E-commerceMicroservices/auth-service/db"
	"github.com/geoo115/E-commerceMicroservices/auth-service/proto"
	"github.com/geoo115/E-commerceMicroservices/auth-service/services"
	"github.com/joho/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load environment variables from .env file if it exists
	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found or error loading it. Using environment variables.")
	}

	if _, err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	if redisClient := cache.InitRedis(); redisClient == nil {
		log.Fatalf("Failed to initialize Redis")
	}

	// Create admin user if it doesn't exist
	services.CreateAdminIfNotExists()

	// Set up gRPC server
	port := os.Getenv("AUTH_SERVICE_PORT")
	if port == "" {
		port = ":50051"
	} else if port[0] != ':' {
		port = ":" + port
	}

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterAuthServiceServer(grpcServer, services.NewAuthServer())
	reflection.Register(grpcServer)

	log.Printf("🚀 Auth gRPC server running on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
