package events

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/geoo115/E-commerceMicroservices/pkg/database/dbtest"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

type fakePublisher struct {
	mu   sync.Mutex
	msgs []Message
	err  error
}

func (f *fakePublisher) Publish(_ context.Context, msgs ...Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.msgs = append(f.msgs, msgs...)
	return nil
}

func TestEventRoundTrip(t *testing.T) {
	ev, err := New(OrderCreated, OrderCreatedData{OrderID: 7, Items: []OrderItem{{ProductID: 1, Quantity: 2}}})
	require.NoError(t, err)
	assert.NotEmpty(t, ev.ID)

	var data OrderCreatedData
	require.NoError(t, ev.Decode(&data))
	assert.Equal(t, uint64(7), data.OrderID)
}

func TestConsumer_RetriesThenDeadLetters(t *testing.T) {
	dlq := &fakePublisher{}
	attempts := 0
	c := &Consumer{
		handler: func(context.Context, Event) error { attempts++; return errors.New("boom") },
		dlq:     dlq,
		log:     discard,
	}
	ev, _ := New(OrderCreated, OrderCreatedData{OrderID: 1})
	value, _ := json.Marshal(ev)

	err := c.process(context.Background(), kafka.Message{Topic: TopicOrders, Key: []byte("1"), Value: value})
	require.NoError(t, err)
	assert.Equal(t, MaxAttempts, attempts)
	require.Len(t, dlq.msgs, 1)
	assert.Equal(t, "orders.dlq", dlq.msgs[0].Topic)
}

func TestConsumer_InvalidEnvelopeGoesToDLQ(t *testing.T) {
	dlq := &fakePublisher{}
	c := &Consumer{handler: func(context.Context, Event) error { return nil }, dlq: dlq, log: discard}

	require.NoError(t, c.process(context.Background(), kafka.Message{Topic: TopicOrders, Value: []byte("not json")}))
	assert.Len(t, dlq.msgs, 1)
}

func TestConsumer_SucceedsAfterTransientError(t *testing.T) {
	dlq := &fakePublisher{}
	attempts := 0
	c := &Consumer{
		handler: func(context.Context, Event) error {
			attempts++
			if attempts == 1 {
				return errors.New("transient")
			}
			return nil
		},
		dlq: dlq,
		log: discard,
	}
	ev, _ := New(OrderCreated, OrderCreatedData{OrderID: 1})
	value, _ := json.Marshal(ev)

	require.NoError(t, c.process(context.Background(), kafka.Message{Topic: TopicOrders, Value: value}))
	assert.Equal(t, 2, attempts)
	assert.Empty(t, dlq.msgs)
}

// --- integration tests (require TEST_DATABASE_URL) ---

func TestRelay_PublishesOnceAndMarksRows(t *testing.T) {
	db := dbtest.New(t, &OutboxMessage{})
	pub := &fakePublisher{}
	relay := NewRelay(db, pub, discard)

	for i := range 3 {
		ev, _ := New(OrderCreated, OrderCreatedData{OrderID: uint64(i)})
		require.NoError(t, Enqueue(db, TopicOrders, "k", ev))
	}

	n, err := relay.Flush(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = relay.Flush(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n, "published rows are not sent again")
	assert.Len(t, pub.msgs, 3)
}

func TestRelay_KeepsRowsWhenBrokerIsDown(t *testing.T) {
	db := dbtest.New(t, &OutboxMessage{})
	pub := &fakePublisher{err: errors.New("broker down")}
	relay := NewRelay(db, pub, discard)

	ev, _ := New(OrderCreated, OrderCreatedData{OrderID: 1})
	require.NoError(t, Enqueue(db, TopicOrders, "1", ev))

	_, err := relay.Flush(context.Background())
	require.Error(t, err)

	pub.err = nil
	n, err := relay.Flush(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n, "the message is retried once the broker is back")
}

func TestProcessOnce_SkipsDuplicates(t *testing.T) {
	db := dbtest.New(t, &ProcessedEvent{})
	ev, _ := New(OrderCreated, OrderCreatedData{OrderID: 1})

	calls := 0
	fn := func(*gorm.DB) error { calls++; return nil }
	require.NoError(t, ProcessOnce(context.Background(), db, "test", ev, fn))
	require.NoError(t, ProcessOnce(context.Background(), db, "test", ev, fn))
	assert.Equal(t, 1, calls)

	// A failed handler does not mark the event as processed.
	ev2, _ := New(OrderCreated, OrderCreatedData{OrderID: 2})
	require.Error(t, ProcessOnce(context.Background(), db, "test", ev2, func(*gorm.DB) error { return errors.New("boom") }))
	require.NoError(t, ProcessOnce(context.Background(), db, "test", ev2, fn))
	assert.Equal(t, 2, calls)
}
