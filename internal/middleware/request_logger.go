package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type requestLogWriter struct {
	http.ResponseWriter

	status int
	bytes  int
	err    error
	route  string
}

// RequestLogger logs the outcome and duration of every HTTP request.
func RequestLogger(logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			response := &requestLogWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
				bytes:          0,
				err:            nil,
				route:          "",
			}

			next.ServeHTTP(response, r)

			attributes := []any{
				"request_id", RequestIDFromContext(r.Context()),
				"method", r.Method,
				"route", requestRoute(response.route, r.Pattern),
				"status", response.status,
				"response_bytes", response.bytes,
				"duration", time.Since(startedAt),
			}
			if response.err != nil {
				attributes = append(attributes, "error", response.err)
			}

			switch {
			case response.status >= http.StatusInternalServerError:
				logger.ErrorContext(r.Context(), "http request", attributes...)

			case response.status >= http.StatusBadRequest:
				logger.WarnContext(r.Context(), "http request", attributes...)

			default:
				logger.InfoContext(r.Context(), "http request", attributes...)
			}
		})
	}
}

func (w *requestLogWriter) WriteHeader(status int) {
	if w.status != http.StatusOK || status == http.StatusOK {
		return
	}

	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *requestLogWriter) Write(body []byte) (int, error) {
	written, err := w.ResponseWriter.Write(body)

	w.bytes += written
	if err != nil {
		w.err = err

		return written, fmt.Errorf("write HTTP response: %w", err)
	}

	return written, nil
}

func (w *requestLogWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *requestLogWriter) RecordError(err error) {
	w.err = err
}

func (w *requestLogWriter) RecordRoute(pattern string) {
	w.route = pattern
}

func recordResponseError(writer http.ResponseWriter, err error) {
	for {
		recorder, ok := writer.(interface{ RecordError(err error) })
		if ok {
			recorder.RecordError(err)

			return
		}

		unwrapper, ok := writer.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return
		}

		writer = unwrapper.Unwrap()
	}
}

func requestRoute(recorded, fallback string) string {
	if recorded != "" {
		return recorded
	}

	if fallback == "" {
		return "unmatched"
	}

	return fallback
}
