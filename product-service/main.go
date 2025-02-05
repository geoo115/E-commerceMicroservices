package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/product-service/db"
	"github.com/geoo115/E-commerceMicroservices/product-service/proto"

	"github.com/geoo115/E-commerceMicroservices/product-service/services"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Initialize database
	db.InitDB()
	defer db.CloseDB()

	// Create gRPC server
	grpcServer := grpc.NewServer()
	productServer := NewProductServer()
	proto.RegisterProductServiceServer(grpcServer, productServer)

	// Start server
	port := os.Getenv("PRODUCT_SERVICE_PORT")
	if port == "" {
		port = "50052"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Product service running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func NewProductServer() *services.ProductServer {
	return &services.ProductServer{}
}
