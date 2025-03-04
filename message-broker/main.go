package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/geoo115/E-commerceMicroservices/message-broker/consumers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/producers"
	"github.com/geoo115/E-commerceMicroservices/message-broker/topics"
)

func main() {
	log.Println("🚀 Starting Message Broker Service...")

	// Define a generic handler that logs the event.
	handler := func(message []byte) {
		log.Printf("🔄 Processing event: %s", string(message))
	}

	// Start consumers with dedicated consumer groups.
	go consumers.ConsumeEvents(topics.OrderPlaced, "order-service-group", handler)
	go consumers.ConsumeEvents(topics.PaymentSuccessful, "payment-service-group", handler)
	go consumers.ConsumeEvents(topics.InventoryUpdated, "inventory-service-group", handler)
	go consumers.ConsumeEvents(topics.ReviewAdded, "review-service-group", handler)
	go consumers.ConsumeEvents(topics.ReviewDeleted, "review-service-group", handler)
	go consumers.ConsumeEvents(topics.WishlistUpdated, "wishlist-service-group", handler)

	// Publish a test event.
	go func() {
		message := []byte("Test order placed event")
		if err := producers.PublishEvent(topics.OrderPlaced, message); err != nil {
			log.Printf("❌ Error publishing event: %v", err)
		}
	}()

	// Wait for termination signal.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("🛑 Shutting down Message Broker Service...")
}
