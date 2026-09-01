package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"singlepage/internal/middleware"
)

func TestBearerTokenStoresTokenInRequestContext(t *testing.T) {
	t.Parallel()

	var captured string

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = middleware.BearerTokenFromContext(r.Context())
	})
	request := newRequest(http.MethodPut, "/api/pages/id", nil)
	request.Header.Set("Authorization", "Bearer secret-token")

	middleware.BearerToken(next).ServeHTTP(httptest.NewRecorder(), request)

	if captured != "secret-token" {
		t.Fatalf("context token = %q, want secret-token", captured)
	}
}

func TestBearerTokenFromContextReturnsEmptyWithoutMiddleware(t *testing.T) {
	t.Parallel()

	request := newRequest(http.MethodGet, "/", nil)
	if token := middleware.BearerTokenFromContext(request.Context()); token != "" {
		t.Fatalf("context token = %q, want empty", token)
	}
}
