package main

import (
	"log"
	"net"
	"os"

	"order-service/db"
	pb "order-service/proto"
	"order-service/services"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load environment variables.
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Initialize the database.
	db.InitDB()
	defer db.CloseDB()

	// Get the service port.
	port := os.Getenv("ORDER_SERVICE_PORT")
	if port == "" {
		port = "50053"
	}

	// Set up the gRPC server.
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, services.NewOrderServer())
	reflection.Register(grpcServer)

	log.Printf("Order Service running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
