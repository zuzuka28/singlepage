package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"singlepage/internal/middleware"
)

func TestWhenAppliesSelectedMiddlewareWhenPredicateMatches(t *testing.T) {
	t.Parallel()

	selectedCalled := false
	selected := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			selectedCalled = true

			next.ServeHTTP(w, r)
		})
	}
	handler := middleware.When(func(*http.Request) bool { return true }, selected)(http.NotFoundHandler())

	handler.ServeHTTP(httptest.NewRecorder(), newRequest(http.MethodGet, "/", nil))

	if !selectedCalled {
		t.Fatal("selected middleware was not called")
	}
}

func TestWhenSkipsSelectedMiddlewareWhenPredicateDoesNotMatch(t *testing.T) {
	t.Parallel()

	selectedCalled := false
	selected := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			selectedCalled = true

			next.ServeHTTP(w, r)
		})
	}
	handler := middleware.When(func(*http.Request) bool { return false }, selected)(http.NotFoundHandler())

	handler.ServeHTTP(httptest.NewRecorder(), newRequest(http.MethodGet, "/", nil))

	if selectedCalled {
		t.Fatal("selected middleware was called")
	}
}
