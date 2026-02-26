package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// tokenBucket implements a simple token bucket rate limiter for a single key.
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// allow checks if a request is allowed and consumes a token if so.
func (b *tokenBucket) allow(now time.Time) bool {
	// Refill tokens based on elapsed time.
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// RateLimiter is an in-memory rate limiter that tracks request rates per agent ID.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[uuid.UUID]*tokenBucket

	maxTokens  float64
	refillRate float64 // tokens per second

	// cleanupInterval controls how often stale buckets are purged.
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewRateLimiter creates a new rate limiter with the given maximum requests per minute.
// A background goroutine periodically cleans up stale entries.
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	maxTokens := float64(requestsPerMinute)
	refillRate := maxTokens / 60.0 // tokens per second

	rl := &RateLimiter{
		buckets:         make(map[uuid.UUID]*tokenBucket),
		maxTokens:       maxTokens,
		refillRate:      refillRate,
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	go rl.cleanup()

	return rl
}

// Allow checks whether the given agent is within their rate limit.
func (rl *RateLimiter) Allow(agentID uuid.UUID) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[agentID]
	if !exists {
		bucket = &tokenBucket{
			tokens:     rl.maxTokens,
			maxTokens:  rl.maxTokens,
			refillRate: rl.refillRate,
			lastRefill: now,
		}
		rl.buckets[agentID] = bucket
	}

	return bucket.allow(now)
}

// cleanup periodically removes stale token buckets (agents that haven't made
// requests in a while and have full buckets).
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for id, bucket := range rl.buckets {
				// If the bucket hasn't been used for 10 minutes, remove it.
				if now.Sub(bucket.lastRefill) > 10*time.Minute {
					delete(rl.buckets, id)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop terminates the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// RateLimitMiddleware returns a Gin middleware that enforces rate limiting per agent.
// It expects the agent_id to be set in the Gin context (by the auth middleware).
// Unauthenticated requests are rate-limited by a zero UUID.
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		agentID := uuid.Nil
		if val, exists := c.Get("agent_id"); exists {
			if id, ok := val.(uuid.UUID); ok {
				agentID = id
			}
		}

		if !limiter.Allow(agentID) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "too many requests, please slow down",
				},
			})
			return
		}

		c.Next()
	}
}
