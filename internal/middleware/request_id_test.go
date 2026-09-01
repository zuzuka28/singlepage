package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"singlepage/internal/middleware"
)

func TestRequestID(t *testing.T) {
	t.Parallel()

	var contextRequestID string

	next := http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		contextRequestID = middleware.RequestIDFromContext(request.Context())
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	middleware.RequestID(next).ServeHTTP(response, request)

	headerRequestID := response.Header().Get("X-Request-ID")
	if headerRequestID == "" || contextRequestID != headerRequestID {
		t.Fatalf("header ID = %q, context ID = %q", headerRequestID, contextRequestID)
	}
}
