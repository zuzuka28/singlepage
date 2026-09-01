package middleware_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"singlepage/internal/middleware"
)

func TestMaxBodyBytesRejectsReadingPastLimit(t *testing.T) {
	t.Parallel()

	var readErr error

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	})
	handler := middleware.MaxBodyBytes(4)(next)

	handler.ServeHTTP(
		httptest.NewRecorder(),
		newRequest(http.MethodPost, "/", strings.NewReader("12345")),
	)

	var maxBytesErr *http.MaxBytesError
	if !errors.As(readErr, &maxBytesErr) {
		t.Fatalf("read error = %v, want *http.MaxBytesError", readErr)
	}
}
