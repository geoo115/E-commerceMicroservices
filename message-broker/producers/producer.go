package producers

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// PublishEvent sends an event to Kafka
func PublishEvent(topic string, message []byte) error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost:9092"})
	if err != nil {
		return err
	}
	defer producer.Close()

	deliveryChan := make(chan kafka.Event)
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, deliveryChan)

	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		log.Printf("❌ Failed to send message to %s: %v", topic, m.TopicPartition.Error)
		return m.TopicPartition.Error
	}

	log.Printf("✅ Message sent to topic %s", topic)
	return nil
}
