// Package events implements asynchronous messaging over Kafka:
// a common event envelope, a transactional outbox for publishing,
// idempotent consumers (inbox) and dead-letter handling.
package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event is the envelope written to every Kafka message.
type Event struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

// New builds an event of the given type with data marshalled as JSON.
func New(eventType string, data any) (Event, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Event{}, fmt.Errorf("marshal %s payload: %w", eventType, err)
	}
	return Event{
		ID:         uuid.NewString(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Data:       raw,
	}, nil
}

// Decode unmarshals the event payload into v.
func (e Event) Decode(v any) error {
	if err := json.Unmarshal(e.Data, v); err != nil {
		return fmt.Errorf("decode %s payload: %w", e.Type, err)
	}
	return nil
}
