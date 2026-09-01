package frontend_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"singlepage/internal/handler/frontend"
)

func TestServeHTTPServesStaticAsset(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeFrontendFile(t, directory, "app.js", "application")
	handler := frontend.NewHandlerForTest(os.DirFS(directory), []byte("fallback"))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, newFrontendRequest(http.MethodGet, "/app.js"))

	assertFrontendResponse(t, response, http.StatusOK, "application")
}

func TestServeHTTPServesIndexForPageRoute(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeFrontendFile(t, directory, "index.html", "SPA INDEX")
	handler := frontend.NewHandlerForTest(os.DirFS(directory), []byte("fallback"))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, newFrontendRequest(http.MethodGet, "/p/0123456789abcdef"))

	assertFrontendResponse(t, response, http.StatusOK, "SPA INDEX")
}

func TestServeHTTPFallsBackToIndexForClientRoute(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeFrontendFile(t, directory, "index.html", "SPA INDEX")
	handler := frontend.NewHandlerForTest(os.DirFS(directory), []byte("fallback"))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, newFrontendRequest(http.MethodGet, "/unknown-client-route"))

	assertFrontendResponse(t, response, http.StatusOK, "SPA INDEX")
}

func TestServeHTTPReturnsFallbackWhenAssetsAreUnavailable(t *testing.T) {
	t.Parallel()

	handler := frontend.NewHandlerForTest(nil, []byte("fallback"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, newFrontendRequest(http.MethodGet, "/"))

	assertFrontendResponse(t, response, http.StatusServiceUnavailable, "fallback")

	if contentType := response.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", contentType)
	}
}

func TestServeHTTPRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	handler := frontend.NewHandlerForTest(nil, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, newFrontendRequest(http.MethodPost, "/"))

	assertFrontendResponse(t, response, http.StatusMethodNotAllowed, "method not allowed\n")

	if allow := response.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("Allow = %q, want GET, HEAD", allow)
	}
}

func writeFrontendFile(t *testing.T, directory, name, contents string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600)
	if err != nil {
		t.Fatalf("write frontend file: %v", err)
	}
}

func assertFrontendResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantBody string,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d", response.Code, wantStatus)
	}

	if response.Body.String() != wantBody {
		t.Fatalf("body = %q, want %q", response.Body.String(), wantBody)
	}
}

func newFrontendRequest(method, target string) *http.Request {
	return httptest.NewRequestWithContext(context.Background(), method, target, nil)
}
