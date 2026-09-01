package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

func BearerAuth(token string) Middleware {
	if token == "" {
		return func(http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeError(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			})
		}
	}

	want := sha256.Sum256([]byte(token))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := parseBearerToken(r.Header.Get("Authorization"))
			if provided == "" {
				writeError(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))

				return
			}

			got := sha256.Sum256([]byte(provided))

			if subtle.ConstantTimeCompare(want[:], got[:]) != 1 {
				writeError(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseBearerToken(header string) string {
	const prefix = "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
