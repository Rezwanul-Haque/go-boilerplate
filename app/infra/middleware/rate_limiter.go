package middleware

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type ipLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPLimiterStore(r rate.Limit, b int) *ipLimiterStore {
	return &ipLimiterStore{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (s *ipLimiterStore) get(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	if l, ok := s.limiters[ip]; ok {
		return l
	}
	l := rate.NewLimiter(s.r, s.b)
	s.limiters[ip] = l
	return l
}

// RateLimit returns a per-IP token bucket middleware.
// r is requests per second; b is the burst size.
// For 5 req/min use: RateLimit(rate.Limit(5.0/60.0), 5)
func RateLimit(r rate.Limit, b int) echo.MiddlewareFunc {
	store := newIPLimiterStore(r, b)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !store.get(ip).Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "too many requests, please try again later",
				})
			}
			return next(c)
		}
	}
}
