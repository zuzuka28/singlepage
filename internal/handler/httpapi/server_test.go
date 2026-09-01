package httpapi_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"singlepage/internal/config"
	"singlepage/internal/handler/httpapi"
	"singlepage/internal/metrics"
)

func TestNewWiresRequestIDLoggingAndMetrics(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	applicationMetrics := metrics.New()
	handler := httpapi.New(
		newFakePageService(),
		config.Config{Protection: config.Protection{MaxRequestBodyBytes: 1024}},
		applicationMetrics,
		logger,
	)
	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/pages/too-short",
		nil,
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("API status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	requestID := response.Header().Get("X-Request-ID")
	if requestID == "" || !strings.Contains(logs.String(), requestID) {
		t.Fatalf("request ID %q was not included in access log %q", requestID, logs.String())
	}

	metricsRequest := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/metrics",
		nil,
	)
	metricsResponse := httptest.NewRecorder()
	applicationMetrics.Handler().ServeHTTP(metricsResponse, metricsRequest)

	metric := `singlepage_api_request_errors_total{method="GET",route="GET /api/pages/{id}",status="400"} 1`
	if !strings.Contains(metricsResponse.Body.String(), metric) {
		t.Fatalf("metrics body does not contain %q:\n%s", metric, metricsResponse.Body.String())
	}

	publicMetricsRequest := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/metrics",
		nil,
	)
	publicMetricsResponse := httptest.NewRecorder()
	handler.ServeHTTP(publicMetricsResponse, publicMetricsRequest)

	if strings.Contains(publicMetricsResponse.Body.String(), "singlepage_api_requests_total") {
		t.Fatal("application server exposed private Prometheus metrics")
	}
}
