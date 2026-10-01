package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OutboxMessage is a pending event stored in the same database (and the same
// transaction) as the state change that produced it. The Relay publishes it
// afterwards, so a state change is never committed without its event and an
// event is never published for a rolled-back change.
type OutboxMessage struct {
	ID          uint64 `gorm:"primaryKey"`
	Topic       string `gorm:"not null"`
	Key         string `gorm:"not null"`
	Payload     []byte `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time
	PublishedAt *time.Time `gorm:"index"`
}

// TableName implements gorm's Tabler.
func (OutboxMessage) TableName() string { return "outbox_messages" }

// Enqueue stores ev in the outbox using tx. Call it inside the transaction
// that performs the corresponding state change.
func Enqueue(tx *gorm.DB, topic, key string, ev Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return tx.Create(&OutboxMessage{Topic: topic, Key: key, Payload: payload}).Error
}

// Relay periodically publishes unpublished outbox messages. Delivery is
// at-least-once: if the process dies after publishing but before marking the
// rows, they are published again, which is why consumers are idempotent.
type Relay struct {
	db        *gorm.DB
	publisher Publisher
	log       *slog.Logger
	interval  time.Duration
	batchSize int
}

// NewRelay returns a relay polling every 200ms.
func NewRelay(db *gorm.DB, publisher Publisher, log *slog.Logger) *Relay {
	return &Relay{db: db, publisher: publisher, log: log, interval: 200 * time.Millisecond, batchSize: 100}
}

// Run publishes outbox messages until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for {
				n, err := r.Flush(ctx)
				if err != nil {
					r.log.Error("outbox relay failed", "error", err)
					break
				}
				if n < r.batchSize {
					break
				}
			}
		}
	}
}

// Flush publishes one batch and returns the number of messages published.
// FOR UPDATE SKIP LOCKED lets several replicas run the relay concurrently.
func (r *Relay) Flush(ctx context.Context) (int, error) {
	var published int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var batch []OutboxMessage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("published_at IS NULL").
			Order("id").
			Limit(r.batchSize).
			Find(&batch).Error; err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}

		msgs := make([]Message, len(batch))
		ids := make([]uint64, len(batch))
		for i, m := range batch {
			msgs[i] = Message{Topic: m.Topic, Key: m.Key, Value: m.Payload}
			ids[i] = m.ID
		}
		if err := r.publisher.Publish(ctx, msgs...); err != nil {
			return err
		}
		for _, m := range batch {
			publishedTotal.WithLabelValues(m.Topic).Inc()
		}
		published = len(batch)
		return tx.Model(&OutboxMessage{}).Where("id IN ?", ids).Update("published_at", time.Now()).Error
	})
	return published, err
}
