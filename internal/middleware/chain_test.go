package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"singlepage/internal/middleware"
)

func TestChainAppliesMiddlewareInDeclarationOrder(t *testing.T) {
	t.Parallel()

	order := make([]string, 0, 3)
	first := recordingMiddleware("first", &order)
	second := recordingMiddleware("second", &order)
	terminal := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		order = append(order, "handler")
	})

	middleware.Chain(terminal, first, second).ServeHTTP(
		httptest.NewRecorder(),
		newRequest(http.MethodGet, "/", nil),
	)

	want := []string{"first", "second", "handler"}
	for index := range want {
		if order[index] != want[index] {
			t.Fatalf("order[%d] = %q, want %q", index, order[index], want[index])
		}
	}
}

func recordingMiddleware(name string, order *[]string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*order = append(*order, name)

			next.ServeHTTP(w, r)
		})
	}
}
