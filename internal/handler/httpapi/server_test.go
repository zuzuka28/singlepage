package httpapi_test

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		true,
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

func TestNewAPIServesSpecificationWithoutFrontend(t *testing.T) {
	t.Parallel()

	handler := httpapi.New(
		newFakePageService(),
		config.Config{Protection: config.Protection{MaxRequestBodyBytes: 1024}},
		metrics.New(),
		slog.New(slog.DiscardHandler),
		false,
	)

	specification := httptest.NewRecorder()
	specificationRequest := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/openapi.json",
		nil,
	)
	handler.ServeHTTP(specification, specificationRequest)

	if specification.Code != http.StatusOK ||
		!strings.Contains(specification.Body.String(), `"openapi":"3.0.3"`) {
		t.Fatalf("spec response = %d %q", specification.Code, specification.Body.String())
	}

	frontend := httptest.NewRecorder()
	frontendRequest := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(frontend, frontendRequest)

	if frontend.Code != http.StatusNotFound {
		t.Fatalf("frontend status = %d, want 404", frontend.Code)
	}
}

func TestServerRunStopsPublicListenerOnCancellation(t *testing.T) {
	t.Parallel()

	server := httpapi.New(
		newFakePageService(),
		serverTestConfig("127.0.0.1:0"),
		metrics.New(),
		slog.New(slog.DiscardHandler),
		false,
	)
	ctx, cancel := context.WithCancel(context.Background())

	result := make(chan error, 1)
	go func() { result <- server.Run(ctx) }()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Run after cancellation: %v", err)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop the public listener")
	}
}

func TestServerRunReportsPublicBindFailure(t *testing.T) {
	t.Parallel()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve address: %v", err)
	}
	defer listener.Close()

	server := httpapi.New(
		newFakePageService(),
		serverTestConfig(listener.Addr().String()),
		metrics.New(),
		slog.New(slog.DiscardHandler),
		false,
	)

	err = server.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "serve public HTTP") {
		t.Fatalf("Run bind error = %v", err)
	}
}

func serverTestConfig(listen string) config.Config {
	return config.Config{
		HTTP: config.HTTP{
			Listen: listen, ReadHeaderTimeout: time.Second, ReadTimeout: time.Second,
			WriteTimeout: time.Second, IdleTimeout: time.Second, ShutdownTimeout: time.Second,
		},
		Protection: config.Protection{MaxRequestBodyBytes: 1024},
	}
}
