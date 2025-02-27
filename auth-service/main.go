package main

import (
	"fmt"
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
	// Load environment variables.
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize the database.
	_, err = db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()

	// Initialize Redis.
	cache.InitRedis()

	// Determine gRPC server port.
	port := os.Getenv("AUTH_SERVICE_PORT")
	if port == "" {
		port = "50051"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterAuthServiceServer(grpcServer, services.NewAuthServer())
	reflection.Register(grpcServer)

	log.Printf("🚀 Auth gRPC server running on port %s", port)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
}
