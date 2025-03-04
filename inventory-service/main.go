package main

import (
	"log"
	"net"
	"os"

	"github.com/geoo115/E-commerceMicroservices/inventory-service/cache"
	"github.com/geoo115/E-commerceMicroservices/inventory-service/db"
	pb "github.com/geoo115/E-commerceMicroservices/inventory-service/proto"
	"github.com/geoo115/E-commerceMicroservices/inventory-service/services"

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

	// Start a consumer for order_placed events using the inventory service's dedicated group.
	go consumers.ConsumeEvents(topics.OrderPlaced, "inventory-service-group", func(message []byte) {
		log.Printf("Inventory Service received order event: %s", string(message))
		// Process the order event. (Implement this handler in services package)
		services.StartOrderPlacedConsumerHandler(message)
	})

	port := os.Getenv("INVENTORY_SERVICE_PORT")
	if port == "" {
		port = "50057"
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
