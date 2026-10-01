package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Handler processes one event. Returning an error triggers a retry; after
// MaxAttempts the message is sent to the dead-letter topic.
type Handler func(ctx context.Context, ev Event) error

// MaxAttempts is the number of times a handler is tried before dead-lettering.
const MaxAttempts = 3

// DeadLetterTopic returns the dead-letter topic for topic.
func DeadLetterTopic(topic string) string { return topic + ".dlq" }

// Consumer reads a topic as part of a consumer group and commits offsets only
// after the handler succeeded (or the message was dead-lettered).
type Consumer struct {
	reader  *kafka.Reader
	handler Handler
	dlq     Publisher
	log     *slog.Logger
	backoff time.Duration
}

// NewConsumer creates a consumer for topic. Each (service, topic) pair gets its
// own consumer group, "<service>.<topic>", so that every service receives
// every event and rebalancing one topic does not pause the others.
func NewConsumer(brokers []string, service, topic string, handler Handler, dlq Publisher, log *slog.Logger) *Consumer {
	groupID := service + "." + topic
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			GroupID:     groupID,
			Topic:       topic,
			StartOffset: kafka.FirstOffset,
			MaxWait:     500 * time.Millisecond,
			// Rebalance when partitions are added, e.g. when another service
			// creates the topic while this consumer is already in the group.
			WatchPartitionChanges: true,
		}),
		handler: handler,
		dlq:     dlq,
		log:     log.With("topic", topic, "group", groupID),
		backoff: 200 * time.Millisecond,
	}
}

// Run consumes messages until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	defer c.reader.Close()
	c.log.Info("consumer started")
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.log.Error("fetch message", "error", err)
			continue
		}

		if err := c.process(ctx, msg); err != nil {
			if ctx.Err() != nil {
				return nil // shutting down; the message will be redelivered
			}
			c.log.Error("message could not be processed or dead-lettered", "offset", msg.Offset, "error", err)
			continue
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil {
			c.log.Error("commit offset", "error", err)
		}
	}
}

func (c *Consumer) process(ctx context.Context, msg kafka.Message) error {
	var ev Event
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		consumedTotal.WithLabelValues(msg.Topic, "unknown", "invalid").Inc()
		return c.deadLetter(ctx, msg, fmt.Errorf("invalid envelope: %w", err))
	}

	var err error
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		if err = c.handler(ctx, ev); err == nil {
			consumedTotal.WithLabelValues(msg.Topic, ev.Type, "ok").Inc()
			return nil
		}
		c.log.Warn("handler failed", "event_id", ev.ID, "type", ev.Type, "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.backoff * time.Duration(1<<(attempt-1))):
		}
	}
	consumedTotal.WithLabelValues(msg.Topic, ev.Type, "dead_lettered").Inc()
	return c.deadLetter(ctx, msg, err)
}

func (c *Consumer) deadLetter(ctx context.Context, msg kafka.Message, cause error) error {
	c.log.Error("sending message to dead-letter topic", "offset", msg.Offset, "error", cause)
	return c.dlq.Publish(ctx, Message{Topic: DeadLetterTopic(msg.Topic), Key: string(msg.Key), Value: msg.Value})
}

// ProcessedEvent records events already handled by a consumer (the "inbox").
type ProcessedEvent struct {
	EventID     string `gorm:"primaryKey"`
	Consumer    string `gorm:"primaryKey"`
	ProcessedAt time.Time
}

// TableName implements gorm's Tabler.
func (ProcessedEvent) TableName() string { return "processed_events" }

// ErrAlreadyProcessed is returned by ProcessOnce for duplicate deliveries.
var ErrAlreadyProcessed = errors.New("event already processed")

// ProcessOnce runs fn in a transaction that also records ev as processed by
// consumer. Redelivered events are skipped, which makes at-least-once delivery
// behave as effectively-once for the state changed inside fn.
func ProcessOnce(ctx context.Context, db *gorm.DB, consumer string, ev Event, fn func(tx *gorm.DB) error) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&ProcessedEvent{EventID: ev.ID, Consumer: consumer, ProcessedAt: time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrAlreadyProcessed
		}
		return fn(tx)
	})
	if errors.Is(err, ErrAlreadyProcessed) {
		return nil
	}
	return err
}
