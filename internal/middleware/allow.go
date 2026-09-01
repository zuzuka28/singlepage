package middleware

import (
	"time"

	"golang.org/x/time/rate"
)

const (
	clientSweepInterval = time.Minute
	clientIdleTTL       = 10 * time.Minute
)

// Allow reports whether client may consume one event at now.
func (l *ClientRateLimiter) Allow(client string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) >= clientSweepInterval {
		l.sweep(now.Add(-clientIdleTTL))
		l.lastSweep = now
	}

	entry := l.clients[client]
	if entry == nil {
		if len(l.clients) >= maxRateLimitClients {
			return false
		}

		entry = &clientLimiter{
			limiter:  rate.NewLimiter(l.limit, l.burst),
			lastSeen: now,
		}
		l.clients[client] = entry
	}

	entry.lastSeen = now

	return entry.limiter.AllowN(now, 1)
}

func (l *ClientRateLimiter) sweep(cutoff time.Time) {
	for key, entry := range l.clients {
		if entry.lastSeen.Before(cutoff) {
			delete(l.clients, key)
		}
	}
}
