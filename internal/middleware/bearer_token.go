package middleware

import (
	"context"
	"net/http"
)

type bearerTokenContextKey struct{}

// BearerToken stores the request Bearer token in context for strict handlers.
func BearerToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := parseBearerToken(r.Header.Get("Authorization"))
		ctx := context.WithValue(r.Context(), bearerTokenContextKey{}, token)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
