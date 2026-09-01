package middleware

import (
	"net/http"
	"time"
)

// RateLimit rejects requests that exceed the per-client token bucket.
func RateLimit(
	limiter *ClientRateLimiter,
	trustProxyHeaders bool,
) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			client := ClientAddress(r, trustProxyHeaders)
			if !limiter.Allow(client, time.Now()) {
				w.Header().Set("Retry-After", "1")
				writeError(w, http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
