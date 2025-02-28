package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/review-service/db"
	pb "github.com/geoo115/E-commerceMicroservices/review-service/proto"
	"github.com/geoo115/E-commerceMicroservices/review-service/services"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	db.InitDB()
	defer db.CloseDB()

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
