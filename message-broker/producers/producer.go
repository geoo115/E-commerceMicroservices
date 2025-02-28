package producers

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka" // note the /v2
)

// PublishEvent sends an event to Kafka on the specified topic.
func PublishEvent(topic string, message []byte) error {
	// Create a new producer using the v2 API.
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "kafka:9092", // use the hostname from your docker-compose (e.g. "kafka")
	})
	if err != nil {
		return err
	}
	defer producer.Close()

	// Create a channel to receive delivery reports.
	deliveryChan := make(chan kafka.Event)

	// Produce the message.
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, deliveryChan)
	if err != nil {
		return err
	}

	// Wait for the delivery report.
	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		log.Printf("❌ Failed to deliver message: %v", m.TopicPartition.Error)
		return m.TopicPartition.Error
	}

	log.Printf("✅ Message delivered to topic %s", topic)
	return nil
}
