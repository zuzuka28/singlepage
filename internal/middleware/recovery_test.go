package middleware_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"singlepage/internal/middleware"
)

var errRecoveredPanic = errors.New("recovered panic was not recorded")

type errorRecordingWriter struct {
	http.ResponseWriter

	err error
}

func (writer *errorRecordingWriter) RecordError(err error) {
	writer.err = err
}

func TestRecoveryDelegatesHealthyRequest(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	middleware.Recovery(next).ServeHTTP(response, newRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestRecoveryReturnsSafeJSONAfterPanic(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive panic")
	})
	response := httptest.NewRecorder()

	middleware.RecoveryWithLogFunc(func(string, ...any) {})(next).ServeHTTP(
		response,
		newRequest(http.MethodGet, "/", nil),
	)

	assertJSONResponse(t, response, http.StatusInternalServerError, "{\"error\":\"internal server error\"}\n")
}

func TestRecoveryLogsPanicEvidence(t *testing.T) {
	t.Parallel()

	var logMessage string

	logger := func(format string, args ...any) {
		logMessage = fmt.Sprintf(format, args...)
	}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive panic")
	})

	response := httptest.NewRecorder()
	middleware.RequestID(middleware.RecoveryWithLogFunc(logger)(next)).ServeHTTP(
		response,
		newRequest(http.MethodGet, "/", nil),
	)

	if !strings.Contains(logMessage, "sensitive panic") {
		t.Fatalf("log message %q does not contain panic evidence", logMessage)
	}

	requestID := response.Header().Get("X-Request-ID")
	if requestID == "" || !strings.Contains(logMessage, "request_id="+requestID) {
		t.Fatalf("log message %q does not contain request ID %q", logMessage, requestID)
	}
}

func TestRecoveryRecordsPanicForRequestLogger(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive panic")
	})
	response := &errorRecordingWriter{
		ResponseWriter: httptest.NewRecorder(),
		err:            errRecoveredPanic,
	}

	middleware.RecoveryWithLogFunc(func(string, ...any) {})(next).ServeHTTP(
		response,
		newRequest(http.MethodGet, "/", nil),
	)

	if response.err == nil || !strings.Contains(response.err.Error(), "sensitive panic") {
		t.Fatalf("recorded error = %v, want panic evidence", response.err)
	}
}
