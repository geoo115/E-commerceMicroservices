package middleware

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/metadata"
)

// RequestIDHeader carries the request ID end to end. It is forwarded to the
// backend services as gRPC metadata so their logs can be correlated.
const RequestIDHeader = "X-Request-ID"

// RequestID assigns every request an ID (or keeps the caller's).
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" || len(id) > 64 {
			id = uuid.NewString()
		}
		c.Header(RequestIDHeader, id)
		ctx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-request-id", id)
		c.Request = c.Request.WithContext(ctx)
		c.Set("request_id", id)
		c.Next()
	}
}

// Logger writes one structured log line per request.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.FullPath() == "/healthz" {
			return
		}
		status := c.Writer.Status()
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		}
		log.Log(c.Request.Context(), level, "http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", c.GetString("request_id"),
			"user_id", UserID(c),
		)
	}
}

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests handled by the gateway.",
	}, []string{"method", "route", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

// Metrics records request count and latency per route template (e.g.
// /api/v1/orders/:id) to keep label cardinality bounded.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		httpRequests.WithLabelValues(c.Request.Method, route, strconv.Itoa(c.Writer.Status())).Inc()
		httpDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}

// Timeout bounds the time spent on a request. The deadline propagates to
// every downstream gRPC call through the request context.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
