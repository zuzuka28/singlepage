package metrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"singlepage/internal/metrics"
)

func TestHandlerExposesPrivateRegistry(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	metrics.New().Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if !strings.Contains(response.Body.String(), "singlepage_api_requests_in_flight 0") {
		t.Fatalf("metrics body does not contain the in-flight gauge:\n%s", response.Body.String())
	}
}
