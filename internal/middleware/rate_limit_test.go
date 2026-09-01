package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"singlepage/internal/middleware"
)

func TestRateLimitRejectsRequestAfterBurstIsExhausted(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewClientRateLimiter(0.000001, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := middleware.RateLimit(limiter, false)(next)
	request := newRequest(http.MethodPost, "/api/pages", nil)
	request.RemoteAddr = testRemoteAddress

	handler.ServeHTTP(httptest.NewRecorder(), request)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assertJSONResponse(t, response, http.StatusTooManyRequests, "{\"error\":\"Too Many Requests\"}\n")

	if retryAfter := response.Header().Get("Retry-After"); retryAfter != "1" {
		t.Fatalf("Retry-After = %q, want 1", retryAfter)
	}
}
