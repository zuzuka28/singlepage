package middleware

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const maxRateLimitClients = 10_000

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ClientRateLimiter struct {
	mu        sync.Mutex
	limit     rate.Limit
	burst     int
	clients   map[string]*clientLimiter
	lastSweep time.Time
}
