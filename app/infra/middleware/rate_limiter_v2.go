package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newTokenBucket(refillRate float64, burst int) *tokenBucket {
	return &tokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

type ipBucketStore struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    float64
	burst   int
}

func newIPBucketStore(rate float64, burst int) *ipBucketStore {
	return &ipBucketStore{
		buckets: make(map[string]*tokenBucket),
		rate:    rate,
		burst:   burst,
	}
}

func (s *ipBucketStore) get(ip string) *tokenBucket {
	s.mu.Lock()
	defer s.mu.Unlock()

	if b, ok := s.buckets[ip]; ok {
		return b
	}
	b := newTokenBucket(s.rate, s.burst)
	s.buckets[ip] = b
	return b
}

// RateLimitV2 returns per-IP token bucket middleware (no third-party rate lib).
// refillRate is tokens per second; burst is max burst size.
// For 5 req/min use: RateLimitV2(5.0/60.0, 5)
func RateLimitV2(refillRate float64, burst int) echo.MiddlewareFunc {
	store := newIPBucketStore(refillRate, burst)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !store.get(ip).allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "too many requests, please try again later",
				})
			}
			return next(c)
		}
	}
}
