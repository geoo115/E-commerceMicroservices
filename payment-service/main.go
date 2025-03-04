package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/payment-service/cache"
	"github.com/geoo115/E-commerceMicroservices/payment-service/db"
	pb "github.com/geoo115/E-commerceMicroservices/payment-service/proto"
	"github.com/geoo115/E-commerceMicroservices/payment-service/services"

	"github.com/geoo115/E-commerceMicroservices/message-broker/consumers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/topics"
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

	cache.InitRedis()

	// Optionally, start a consumer for order_placed events using the payment service's dedicated group.
	go consumers.ConsumeEvents(topics.OrderPlaced, "payment-service-group", func(message []byte) {
		log.Printf("Payment Service received order_placed event: %s", string(message))
		// Process the event to initiate payment processing.
		services.HandleOrderPlacedForPayment(message)
	})

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
