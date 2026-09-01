package middleware_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"singlepage/internal/middleware"
)

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&output, nil))
	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.Pattern = "/api/"

		if recorder, ok := writer.(interface{ RecordRoute(pattern string) }); ok {
			recorder.RecordRoute("GET /api/pages/{id}")
		}

		if recorder, ok := writer.(interface{ RecordError(err error) }); ok {
			recorder.RecordError(io.ErrUnexpectedEOF)
		}

		writer.WriteHeader(http.StatusInternalServerError)
	})
	handler := middleware.RequestID(middleware.RequestLogger(logger)(next))
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/pages/id", nil)

	handler.ServeHTTP(httptest.NewRecorder(), request)

	logged := output.String()
	for _, value := range []string{
		`"level":"ERROR"`,
		`"request_id":`,
		`"route":"GET /api/pages/{id}"`,
		`"status":500`,
		`"error":"unexpected EOF"`,
	} {
		if !strings.Contains(logged, value) {
			t.Fatalf("log %q does not contain %q", logged, value)
		}
	}
}
