package middleware

import "net/http"

// When applies middleware only to requests accepted by predicate.
func When(predicate func(*http.Request) bool, selected Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		protected := selected(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if predicate(r) {
				protected.ServeHTTP(w, r)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
