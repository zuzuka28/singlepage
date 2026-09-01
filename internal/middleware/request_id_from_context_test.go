package middleware_test

import (
	"context"
	"testing"

	"singlepage/internal/middleware"
)

func TestRequestIDFromContext(t *testing.T) {
	t.Parallel()

	if requestID := middleware.RequestIDFromContext(context.Background()); requestID != "" {
		t.Fatalf("request ID = %q, want empty", requestID)
	}
}
