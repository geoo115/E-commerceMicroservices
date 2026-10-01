package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/geoo115/E-commerceMicroservices/pkg/config"
)

// BrokersFromEnv returns the brokers listed in KAFKA_BROKERS.
func BrokersFromEnv() []string {
	return config.List("KAFKA_BROKERS", []string{"localhost:9092"})
}

// Message is a single record to publish.
type Message struct {
	Topic string
	Key   string
	Value []byte
}

// Publisher writes messages to the message broker.
type Publisher interface {
	Publish(ctx context.Context, msgs ...Message) error
}

// KafkaPublisher is a Publisher backed by a kafka-go Writer.
type KafkaPublisher struct {
	w *kafka.Writer
}

// NewKafkaPublisher returns a publisher that waits for all in-sync replicas to acknowledge.
func NewKafkaPublisher(brokers []string) *KafkaPublisher {
	return &KafkaPublisher{w: &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.Hash{}, // same key -> same partition -> ordered
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: 10 * time.Millisecond,
	}}
}

// Publish writes msgs synchronously.
func (p *KafkaPublisher) Publish(ctx context.Context, msgs ...Message) error {
	km := make([]kafka.Message, len(msgs))
	for i, m := range msgs {
		km[i] = kafka.Message{Topic: m.Topic, Key: []byte(m.Key), Value: m.Value}
	}
	return p.w.WriteMessages(ctx, km...)
}

// Close flushes pending writes and closes the writer.
func (p *KafkaPublisher) Close() error { return p.w.Close() }

// EnsureTopics creates the given topics (and their dead-letter topics) if they
// do not exist yet, retrying until the broker is reachable or ctx is done.
func EnsureTopics(ctx context.Context, brokers []string, log *slog.Logger, topics ...string) error {
	partitions := config.Int("KAFKA_TOPIC_PARTITIONS", 3)
	replication := config.Int("KAFKA_REPLICATION_FACTOR", 1)

	configs := make([]kafka.TopicConfig, 0, len(topics)*2)
	for _, t := range topics {
		for _, name := range []string{t, DeadLetterTopic(t)} {
			configs = append(configs, kafka.TopicConfig{
				Topic:             name,
				NumPartitions:     partitions,
				ReplicationFactor: replication,
			})
		}
	}

	for {
		err := createTopics(brokers[0], configs)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("ensure kafka topics: %w", err)
		}
		log.Warn("kafka not ready, retrying", "error", err)
		select {
		case <-ctx.Done():
			return fmt.Errorf("ensure kafka topics: %w", err)
		case <-time.After(2 * time.Second):
		}
	}
}

func createTopics(broker string, configs []kafka.TopicConfig) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	cc, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer cc.Close()

	if err := cc.CreateTopics(configs...); err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
		return err
	}

	// Topic creation is asynchronous: wait until every partition has a leader,
	// otherwise a consumer group joining right now could get no partitions.
	names := make([]string, len(configs))
	for i, c := range configs {
		names[i] = c.Topic
	}
	partitions, err := conn.ReadPartitions(names...)
	if err != nil {
		return err
	}
	ready := make(map[string]int)
	for _, p := range partitions {
		if p.Leader.Host != "" {
			ready[p.Topic]++
		}
	}
	for _, c := range configs {
		if ready[c.Topic] < c.NumPartitions {
			return fmt.Errorf("topic %s is not ready yet", c.Topic)
		}
	}
	return nil
}
