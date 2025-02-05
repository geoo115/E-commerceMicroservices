package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/inventory-service/db"
	pb "github.com/geoo115/E-commerceMicroservices/inventory-service/proto"
	"github.com/geoo115/E-commerceMicroservices/inventory-service/services"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load environment variables from .env.
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Initialize database.
	db.InitDB()
	defer db.CloseDB()

	// Get the service port from environment variables.
	port := os.Getenv("INVENTORY_SERVICE_PORT")
	if port == "" {
		port = "50057" // default port if not set
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterInventoryServiceServer(grpcServer, services.NewInventoryServer())
	reflection.Register(grpcServer)

	log.Printf("Inventory Service running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
