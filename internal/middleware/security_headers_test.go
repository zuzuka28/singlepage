package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"singlepage/internal/middleware"
)

func TestSecurityHeadersSetsBaselineProtections(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, newRequest(http.MethodGet, "/", nil))

	want := map[string]string{
		"Cache-Control":                "no-store",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "camera=(), geolocation=(), microphone=(), payment=(), usb=()",
		"Referrer-Policy":              "no-referrer",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
	}
	for name, value := range want {
		if got := response.Header().Get(name); got != value {
			t.Fatalf("%s = %q, want %q", name, got, value)
		}
	}

	if policy := response.Header().Get("Content-Security-Policy"); policy == "" {
		t.Fatal("Content-Security-Policy is empty")
	}
}

func TestSecurityHeadersSetsHSTSForForwardedHTTPS(t *testing.T) {
	t.Parallel()

	request := newRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")

	response := httptest.NewRecorder()

	middleware.SecurityHeaders(http.NotFoundHandler()).ServeHTTP(response, request)

	if value := response.Header().Get("Strict-Transport-Security"); value != "max-age=31536000" {
		t.Fatalf("Strict-Transport-Security = %q, want max-age=31536000", value)
	}
}

func TestSecurityHeadersPreventsIndexingPageRoutes(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	middleware.SecurityHeaders(http.NotFoundHandler()).ServeHTTP(
		response,
		newRequest(http.MethodGet, "/p/0123456789abcdef", nil),
	)

	if value := response.Header().Get("X-Robots-Tag"); value != "noindex, nofollow, noarchive" {
		t.Fatalf("X-Robots-Tag = %q, want noindex, nofollow, noarchive", value)
	}
}
