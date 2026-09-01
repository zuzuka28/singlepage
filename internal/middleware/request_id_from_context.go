package middleware

import "context"

// RequestIDFromContext returns the identifier assigned to the request.
func RequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	if !ok {
		return ""
	}

	return requestID
}
