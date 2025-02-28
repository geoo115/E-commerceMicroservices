package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/cart-service/cache"
	"github.com/geoo115/E-commerceMicroservices/cart-service/db"
	pb "github.com/geoo115/E-commerceMicroservices/cart-service/proto"
	"github.com/geoo115/E-commerceMicroservices/cart-service/services"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	// Load the .env file (adjust the path if needed)
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Initialize the database.
	db.InitDB()
	defer db.CloseDB()

	// Initialize Redis.
	cache.InitRedis()

	// Get the service port from environment variables.
	port := os.Getenv("CART_SERVICE_PORT")
	if port == "" {
		port = "50054"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCartServiceServer(grpcServer, services.NewCartServer())

	log.Printf("Cart Service running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
