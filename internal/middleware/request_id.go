package middleware

import (
	"context"
	"crypto/rand"
	"net/http"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

// RequestID assigns a cryptographically random identifier to every request.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := rand.Text()
		w.Header().Set(requestIDHeader, requestID)

		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
