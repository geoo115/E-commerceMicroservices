package producers

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// PublishEvent sends an event to Kafka on the specified topic.
func PublishEvent(topic string, message []byte) error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "kafka:9092", // Adjust as needed.
	})
	if err != nil {
		return err
	}
	defer producer.Close()

	deliveryChan := make(chan kafka.Event)
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, deliveryChan)
	if err != nil {
		return err
	}

	e := <-deliveryChan
	m := e.(*kafka.Message)
	if m.TopicPartition.Error != nil {
		log.Printf("❌ Failed to deliver message: %v", m.TopicPartition.Error)
		return m.TopicPartition.Error
	}

	log.Printf("✅ Message delivered to topic %s", topic)
	return nil
}
