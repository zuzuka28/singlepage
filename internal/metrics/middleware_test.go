package metrics_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"singlepage/internal/metrics"
)

func TestMiddlewareRecordsMatchedRequestAndPreservesWriter(t *testing.T) {
	t.Parallel()

	collector := metrics.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /pages/{id}", func(writer http.ResponseWriter, _ *http.Request) {
		if _, ok := writer.(interface{ Unwrap() http.ResponseWriter }); !ok {
			t.Error("instrumented response writer does not expose Unwrap")
		}

		writer.WriteHeader(http.StatusCreated)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/pages/example", nil)
	collector.Middleware(mux).ServeHTTP(response, request)
	body := scrape(t, collector)

	assertContains(
		t,
		body,
		`singlepage_api_requests_total{method="GET",route="GET /pages/{id}",status="201"} 1`,
	)
	assertContains(
		t,
		body,
		`singlepage_api_request_duration_seconds_count{method="GET",route="GET /pages/{id}",status="201"} 1`,
	)
	assertContains(t, body, "singlepage_api_requests_in_flight 0")
}

func TestMiddlewareRecordsErrorsAndUnmatchedRoutes(t *testing.T) {
	t.Parallel()

	collector := metrics.New()
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing", nil)
	collector.Middleware(http.NewServeMux()).ServeHTTP(response, request)
	body := scrape(t, collector)

	assertContains(
		t,
		body,
		`singlepage_api_request_errors_total{method="GET",route="unmatched",status="404"} 1`,
	)
}

func TestMiddlewareInfersSuccessfulStatusAndBoundsMethods(t *testing.T) {
	t.Parallel()

	collector := metrics.New()
	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.Pattern = "CUSTOM /pages"

		_, err := io.WriteString(writer, "ok")
		if err != nil {
			t.Errorf("write response: %v", err)
		}
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), "CUSTOM", "/pages", nil)
	collector.Middleware(next).ServeHTTP(response, request)
	body := scrape(t, collector)

	assertContains(
		t,
		body,
		`singlepage_api_requests_total{method="OTHER",route="CUSTOM /pages",status="200"} 1`,
	)
}

func scrape(t *testing.T, collector *metrics.Metrics) string {
	t.Helper()

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	collector.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", response.Code, http.StatusOK)
	}

	return response.Body.String()
}

func assertContains(t *testing.T, body, metric string) {
	t.Helper()

	if !strings.Contains(body, metric) {
		t.Fatalf("metrics body does not contain %q:\n%s", metric, body)
	}
}
