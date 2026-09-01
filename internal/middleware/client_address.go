package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientAddress resolves the rate-limit key for a request.
func ClientAddress(r *http.Request, trustProxyHeaders bool) string {
	address := r.RemoteAddr

	if trustProxyHeaders {
		forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")

		candidate := strings.TrimSpace(forwarded[len(forwarded)-1])

		if net.ParseIP(candidate) != nil {
			address = candidate
		}
	}

	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return host
	}

	return address
}
