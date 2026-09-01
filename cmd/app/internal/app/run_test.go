//go:build wails

package app //nolint:testpackage // The asset handler is intentionally package-private.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNativeAssetHandlerUsesIndexAndSecurityHeadersForPageRoute(t *testing.T) {
	t.Parallel()

	assetsHandler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" {
			http.NotFound(response, request)

			return
		}

		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("index"))
	})
	handler := nativeAssetHandlerWith(assetsHandler)
	root := httptest.NewRecorder()
	rootRequest := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler.ServeHTTP(root, rootRequest)

	page := httptest.NewRecorder()
	pageRequest := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/p/0123456789abcdef", nil,
	)
	handler.ServeHTTP(page, pageRequest)

	if root.Code != http.StatusOK {
		t.Fatalf("root status = %d, want %d", root.Code, http.StatusOK)
	}

	if page.Code != root.Code || page.Body.String() != root.Body.String() {
		t.Fatalf("page route did not use index response: root=%d page=%d", root.Code, page.Code)
	}

	if page.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("native response is missing Content-Security-Policy")
	}
}
