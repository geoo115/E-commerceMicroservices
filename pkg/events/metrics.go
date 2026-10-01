package events

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	publishedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "events_published_total",
		Help: "Events published from the outbox, by topic.",
	}, []string{"topic"})

	consumedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "events_consumed_total",
		Help: "Events consumed, by topic, event type and result (ok, invalid, dead_lettered).",
	}, []string{"topic", "type", "result"})
)
