package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter is an in-memory, per-client-IP token bucket. It protects
// sensitive endpoints (login, signup, verification) from brute force. With
// several gateway replicas a shared store such as Redis would be needed.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*client
	limit    rate.Limit
	burst    int
	lastScan time.Time
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter allows perMinute requests per minute per IP, with bursts of burst.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	return &RateLimiter{
		clients:  make(map[string]*client),
		limit:    rate.Limit(float64(perMinute) / 60),
		burst:    burst,
		lastScan: time.Now(),
	}
}

// Middleware rejects requests over the limit with 429.
func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.allow(c.ClientIP()) {
			c.Header("Retry-After", "60")
			abort(c, http.StatusTooManyRequests, "RESOURCE_EXHAUSTED", "too many requests, slow down")
			return
		}
		c.Next()
	}
}

func (r *RateLimiter) allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Sub(r.lastScan) > time.Minute { // drop idle clients so the map does not grow forever
		for k, cl := range r.clients {
			if now.Sub(cl.lastSeen) > 10*time.Minute {
				delete(r.clients, k)
			}
		}
		r.lastScan = now
	}

	cl, ok := r.clients[ip]
	if !ok {
		cl = &client{limiter: rate.NewLimiter(r.limit, r.burst)}
		r.clients[ip] = cl
	}
	cl.lastSeen = now
	return cl.limiter.Allow()
}
