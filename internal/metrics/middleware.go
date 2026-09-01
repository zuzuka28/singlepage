package metrics

import (
	"net/http"
	"strconv"
	"time"
)

const (
	unmatchedRoute = "unmatched"
	otherMethod    = "OTHER"
)

// Middleware records request counts, failures, duration, and concurrency.
func (metrics *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		response := &responseWriter{
			ResponseWriter: writer,
			status:         http.StatusOK,
			wroteHeader:    false,
			route:          "",
		}

		metrics.inFlight.Inc()
		defer metrics.inFlight.Dec()

		next.ServeHTTP(response, request)

		method := methodLabel(request.Method)
		route := routeLabel(response.route, request.Pattern)
		status := strconv.Itoa(response.status)
		metrics.requests.WithLabelValues(method, route, status).Inc()
		metrics.duration.WithLabelValues(method, route, status).Observe(time.Since(startedAt).Seconds())

		if response.status >= http.StatusBadRequest {
			metrics.errors.WithLabelValues(method, route, status).Inc()
		}
	})
}

func methodLabel(method string) string {
	switch method {
	case http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace:
		return method

	default:
		return otherMethod
	}
}

func routeLabel(recorded, fallback string) string {
	if recorded != "" {
		return recorded
	}

	if fallback == "" {
		return unmatchedRoute
	}

	return fallback
}
