package producers

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// PublishEvent sends an event to Kafka on the specified topic.
func PublishEvent(topic string, message []byte) error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "kafka_broker:9092",
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
		log.Printf("❌ Failed to send message to %s: %v", topic, m.TopicPartition.Error)
		return m.TopicPartition.Error
	}

	log.Printf("✅ Message sent to topic %s", topic)
	return nil
}
