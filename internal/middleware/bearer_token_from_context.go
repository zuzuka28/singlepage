package middleware

import "context"

// BearerTokenFromContext returns the token extracted by BearerToken middleware.
func BearerTokenFromContext(ctx context.Context) string {
	token, ok := ctx.Value(bearerTokenContextKey{}).(string)
	if !ok {
		return ""
	}

	return token
}
