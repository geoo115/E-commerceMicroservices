package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/payment-service/db"
	pb "github.com/geoo115/E-commerceMicroservices/payment-service/proto"
	"github.com/geoo115/E-commerceMicroservices/payment-service/services"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load environment variables from .env file (adjust path if needed)
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Initialize the database
	db.InitDB()
	defer db.CloseDB()

	// Get the service port from environment variables
	port := os.Getenv("PAYMENT_SERVICE_PORT")
	if port == "" {
		port = "50055"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, services.NewPaymentServer())
	reflection.Register(grpcServer)

	log.Printf("Payment Service running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
