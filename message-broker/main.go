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

	// Set up a channel to handle OS interrupts for graceful shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Define an event handler for incoming Kafka messages.
	handler := func(message []byte) {
		log.Printf("🔄 Processing event: %s", string(message))
		// Here you can add code to process the event, e.g. updating services.
	}

	// Start consumer in a goroutine.
	go func() {
		consumers.ConsumeEvents(topics.OrderPlaced, handler)
	}()

	// Example: Publish an event (for testing purposes).
	go func() {
		message := []byte("Test order placed event")
		if err := producers.PublishEvent(topics.OrderPlaced, message); err != nil {
			log.Printf("❌ Error publishing event: %v", err)
		}
	}()

	// Block main thread until a termination signal is received.
	<-sigChan
	log.Println("🛑 Shutting down Message Broker Service...")
}
