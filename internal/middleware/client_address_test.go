package middleware_test

import (
	"net/http"
	"testing"

	"singlepage/internal/middleware"
)

func TestClientAddressUsesRemoteHostByDefault(t *testing.T) {
	t.Parallel()

	request := newRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = testRemoteAddress
	request.Header.Set("X-Forwarded-For", "198.51.100.7")

	if address := middleware.ClientAddress(request, false); address != "192.0.2.10" {
		t.Fatalf("address = %q, want 192.0.2.10", address)
	}
}

func TestClientAddressUsesLastTrustedForwardedAddress(t *testing.T) {
	t.Parallel()

	request := newRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = testRemoteAddress
	request.Header.Set("X-Forwarded-For", "203.0.113.8, 198.51.100.7")

	if address := middleware.ClientAddress(request, true); address != "198.51.100.7" {
		t.Fatalf("address = %q, want 198.51.100.7", address)
	}
}

func TestClientAddressIgnoresInvalidForwardedAddress(t *testing.T) {
	t.Parallel()

	request := newRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = testRemoteAddress
	request.Header.Set("X-Forwarded-For", "not-an-ip")

	if address := middleware.ClientAddress(request, true); address != "192.0.2.10" {
		t.Fatalf("address = %q, want 192.0.2.10", address)
	}
}
