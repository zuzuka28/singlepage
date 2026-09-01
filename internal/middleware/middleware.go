package middleware

import (
	"net/http"
	"slices"
)

// Middleware decorates an HTTP handler with cross-cutting server behavior.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware in declaration order: the first item is outermost.
func Chain(handler http.Handler, middleware ...Middleware) http.Handler {
	for _, current := range slices.Backward(middleware) {
		handler = current(handler)
	}

	return handler
}
