package main

import (
	"log"
	"net"
	"os"

	"review-service/db"
	pb "review-service/proto"
	"review-service/services"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load environment variables from .env (adjust path if needed)
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Initialize the database
	db.InitDB()
	defer db.CloseDB()

	// Get the service port.
	port := os.Getenv("REVIEW_SERVICE_PORT")
	if port == "" {
		port = "50056"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterReviewServiceServer(grpcServer, services.NewReviewServer())
	reflection.Register(grpcServer)

	log.Printf("Review Service running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
