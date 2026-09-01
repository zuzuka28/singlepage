package middleware

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// NewClientRateLimiter creates a bounded per-client limiter registry.
func NewClientRateLimiter(eventsPerSecond float64, burst int) *ClientRateLimiter {
	return &ClientRateLimiter{
		mu:        sync.Mutex{},
		limit:     rate.Limit(eventsPerSecond),
		burst:     burst,
		clients:   make(map[string]*clientLimiter),
		lastSweep: time.Now(),
	}
}
