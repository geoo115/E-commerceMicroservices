package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/order-service/cache"
	"github.com/geoo115/E-commerceMicroservices/order-service/db"
	pb "github.com/geoo115/E-commerceMicroservices/order-service/proto"
	"github.com/geoo115/E-commerceMicroservices/order-service/services"

	"github.com/geoo115/E-commerceMicroservices/message-broker/consumers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/topics"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	db.InitDB()
	defer db.CloseDB()

	cache.InitRedis()

	// Start a consumer for payment_successful events using the order service's dedicated group.
	go consumers.ConsumeEvents(topics.PaymentSuccessful, "order-service-group", func(message []byte) {
		log.Printf("Order Service received payment_successful event: %s", string(message))
		// Process the event to update order status.
		services.HandlePaymentSuccessfulEvent(message)
	})

	port := os.Getenv("ORDER_SERVICE_PORT")
	if port == "" {
		port = "50053"
	}

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
