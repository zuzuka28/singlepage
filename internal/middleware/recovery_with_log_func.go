package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

type recoveredPanicError struct {
	value any
}

func (err recoveredPanicError) Error() string {
	return fmt.Sprintf("recovered HTTP panic: %v", err.value)
}

// RecoveryWithLogFunc creates recovery middleware with an injectable logger.
func RecoveryWithLogFunc(logf func(string, ...any)) Middleware {
	if logf == nil {
		logf = defaultPanicLogf
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := RequestIDFromContext(r.Context())

			defer func() {
				if recovered := recover(); recovered != nil {
					recoveryError := recoveredPanicError{value: recovered}
					recordResponseError(w, recoveryError)
					logf(
						"recovered HTTP panic: request_id=%s panic=%v\n%s",
						requestID,
						recovered,
						debug.Stack(),
					)
					writeError(w, http.StatusInternalServerError, "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
