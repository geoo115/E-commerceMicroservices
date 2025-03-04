package consumers

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ConsumeEvents listens to a Kafka topic using the specified consumer group
// and processes each message using the provided handler.
func ConsumeEvents(topic string, groupID string, handler func([]byte)) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "kafka_broker:9092", // Adjust as needed.
		"group.id":          groupID,             // Use the provided group ID.
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("❌ Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	err = consumer.Subscribe(topic, nil)
	if err != nil {
		log.Fatalf("❌ Failed to subscribe to topic %s: %v", topic, err)
	}
	log.Printf("✅ Listening for messages on topic: %s (Group: %s)", topic, groupID)

	// Setup signal handling for graceful shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	run := true
	for run {
		select {
		case <-sigChan:
			log.Println("🛑 Shutting down consumer gracefully...")
			run = false
		default:
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				log.Printf("📥 Received message from %s: %s", topic, string(msg.Value))
				handler(msg.Value)
			} else {
				log.Printf("❌ Consumer error: %v", err)
			}
		}
	}
}
