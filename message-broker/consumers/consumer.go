// message-broker/consumers/consumer.go
package consumers

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// ConsumeEvents listens to a Kafka topic and processes messages
func ConsumeEvents(topic string, handler func([]byte)) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "kafka_broker:9092",
		"group.id":          "order-group",
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

	log.Printf("✅ Listening for messages on topic: %s", topic)

	for {
		msg, err := consumer.ReadMessage(-1)
		if err == nil {
			log.Printf("📥 Received message from %s: %s", topic, string(msg.Value))
			handler(msg.Value)
		} else {
			log.Printf("❌ Consumer error: %v", err)
		}
	}
}
