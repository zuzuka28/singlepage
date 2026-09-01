package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const testID = "01234567-89ab-cdef-0123-456789abcdef"
const rotatedTestID = "fedcba98-7654-3210-fedc-ba9876543210"

func newTestServer(t *testing.T, frontend fs.FS) *Server {
	t.Helper()
	return newTestServerWithConfig(t, frontend, DefaultConfig())
}

func newTestServerWithConfig(t *testing.T, frontend fs.FS, config Config) *Server {
	t.Helper()
	s, err := OpenWithConfig(context.Background(), filepath.Join(t.TempDir(), "data.db"), frontend, []byte("fallback"), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func request(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func createPage(t *testing.T, handler http.Handler) {
	t.Helper()
	response := request(t, handler, http.MethodPost, "/api/pages", map[string]any{
		"id": testID, "salt": []byte("salt-one"), "ciphertext": []byte("cipher-one"), "writeToken": "secret-token",
	}, "")
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestCreateAndReadOpaquePage(t *testing.T) {
	server := newTestServer(t, nil)
	handler := server.Handler()
	createPage(t, handler)

	response := request(t, handler, http.MethodGet, "/api/pages/"+testID, nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", response.Code, response.Body.String())
	}
	var page pageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Revision != 1 || string(page.Salt) != "salt-one" || string(page.Ciphertext) != "cipher-one" {
		t.Fatalf("unexpected page: %+v", page)
	}
	// Read responses must never expose write authentication data.
	if bytes.Contains(response.Body.Bytes(), []byte("secret-token")) || bytes.Contains(response.Body.Bytes(), []byte("writeToken")) {
		t.Fatal("GET exposed write authorization")
	}
	var storedHash []byte
	if err := server.db.QueryRow(`SELECT write_token_hash FROM pages WHERE id = ?`, testID).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if len(storedHash) != sha256.Size || bytes.Equal(storedHash, []byte("secret-token")) {
		t.Fatalf("write token was not stored as a SHA-256 hash: %x", storedHash)
	}
}

func TestCreateRejectsDuplicateID(t *testing.T) {
	handler := newTestServer(t, nil).Handler()
	createPage(t, handler)
	response := request(t, handler, http.MethodPost, "/api/pages", map[string]any{
		"id": testID, "salt": []byte("salt"), "ciphertext": []byte("cipher"), "writeToken": "other",
	}, "")
	if response.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", response.Code)
	}
}

func TestUpdateRequiresTokenAndUsesRevisionCAS(t *testing.T) {
	handler := newTestServer(t, nil).Handler()
	createPage(t, handler)
	body := map[string]any{"expectedRevision": 1, "ciphertext": []byte("cipher-two")}

	if response := request(t, handler, http.MethodPut, "/api/pages/"+testID, body, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want 401", response.Code)
	}
	if response := request(t, handler, http.MethodPut, "/api/pages/"+testID, body, "wrong"); response.Code != http.StatusForbidden {
		t.Fatalf("wrong token status = %d, want 403", response.Code)
	}
	if response := request(t, handler, http.MethodPut, "/api/pages/"+testID, body, "secret-token"); response.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", response.Code, response.Body.String())
	}
	if response := request(t, handler, http.MethodPut, "/api/pages/"+testID, body, "secret-token"); response.Code != http.StatusConflict {
		t.Fatalf("stale update status = %d, want 409", response.Code)
	}

	response := request(t, handler, http.MethodGet, "/api/pages/"+testID, nil, "")
	var page pageResponse
	_ = json.Unmarshal(response.Body.Bytes(), &page)
	if page.Revision != 2 || string(page.Ciphertext) != "cipher-two" {
		t.Fatalf("unexpected updated page: %+v", page)
	}
}

func TestUpdateCanRotateSaltAndWriteToken(t *testing.T) {
	handler := newTestServer(t, nil).Handler()
	createPage(t, handler)
	response := request(t, handler, http.MethodPut, "/api/pages/"+testID, map[string]any{
		"expectedRevision": 1,
		"ciphertext":       []byte("rotated-cipher"),
		"salt":             []byte("rotated-salt"),
		"newWriteToken":    "new-secret-token",
	}, "secret-token")
	if response.Code != http.StatusOK {
		t.Fatalf("rotate status = %d, body = %s", response.Code, response.Body.String())
	}
	body := map[string]any{"expectedRevision": 2, "ciphertext": []byte("next")}
	if response := request(t, handler, http.MethodPut, "/api/pages/"+testID, body, "secret-token"); response.Code != http.StatusForbidden {
		t.Fatalf("old token status = %d, want 403", response.Code)
	}
	if response := request(t, handler, http.MethodPut, "/api/pages/"+testID, body, "new-secret-token"); response.Code != http.StatusOK {
		t.Fatalf("new token status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestRotatePageReplacesOldIDAtomically(t *testing.T) {
	handler := newTestServer(t, nil).Handler()
	createPage(t, handler)
	response := request(t, handler, http.MethodPost, "/api/pages/"+testID+"/rotate", map[string]any{
		"newId": rotatedTestID, "salt": []byte("new-salt"), "ciphertext": []byte("new-cipher"), "newWriteToken": "new-token",
	}, "secret-token")
	if response.Code != http.StatusCreated {
		t.Fatalf("rotate status = %d, body = %s", response.Code, response.Body.String())
	}

	if response := request(t, handler, http.MethodGet, "/api/pages/"+testID, nil, ""); response.Code != http.StatusNotFound {
		t.Fatalf("old page GET status = %d, want 404", response.Code)
	}
	response = request(t, handler, http.MethodGet, "/api/pages/"+rotatedTestID, nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("new page GET status = %d, body = %s", response.Code, response.Body.String())
	}
	var page pageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Revision != 1 || string(page.Salt) != "new-salt" || string(page.Ciphertext) != "new-cipher" {
		t.Fatalf("unexpected rotated page: %+v", page)
	}

	body := map[string]any{"expectedRevision": 1, "ciphertext": []byte("stale-write")}
	if response := request(t, handler, http.MethodPut, "/api/pages/"+testID, body, "secret-token"); response.Code != http.StatusNotFound {
		t.Fatalf("old page update status = %d, want 404", response.Code)
	}
	if response := request(t, handler, http.MethodPut, "/api/pages/"+rotatedTestID, body, "secret-token"); response.Code != http.StatusForbidden {
		t.Fatalf("old token on new page status = %d, want 403", response.Code)
	}
	if response := request(t, handler, http.MethodPut, "/api/pages/"+rotatedTestID, body, "new-token"); response.Code != http.StatusOK {
		t.Fatalf("new page update status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestRotatePageInvalidTokenLeavesOldPageIntact(t *testing.T) {
	handler := newTestServer(t, nil).Handler()
	createPage(t, handler)
	response := request(t, handler, http.MethodPost, "/api/pages/"+testID+"/rotate", map[string]any{
		"newId": rotatedTestID, "salt": []byte("new-salt"), "ciphertext": []byte("new-cipher"), "newWriteToken": "new-token",
	}, "wrong-token")
	if response.Code != http.StatusForbidden {
		t.Fatalf("invalid token status = %d, want 403", response.Code)
	}
	if response := request(t, handler, http.MethodGet, "/api/pages/"+testID, nil, ""); response.Code != http.StatusOK {
		t.Fatalf("old page GET status = %d, want 200", response.Code)
	}
	if response := request(t, handler, http.MethodGet, "/api/pages/"+rotatedTestID, nil, ""); response.Code != http.StatusNotFound {
		t.Fatalf("new page GET status = %d, want 404", response.Code)
	}
}

func TestRotatePageConflictLeavesOldPageIntact(t *testing.T) {
	handler := newTestServer(t, nil).Handler()
	createPage(t, handler)
	response := request(t, handler, http.MethodPost, "/api/pages", map[string]any{
		"id": rotatedTestID, "salt": []byte("occupied-salt"), "ciphertext": []byte("occupied-cipher"), "writeToken": "occupied-token",
	}, "")
	if response.Code != http.StatusCreated {
		t.Fatalf("create conflicting page status = %d, body = %s", response.Code, response.Body.String())
	}
	response = request(t, handler, http.MethodPost, "/api/pages/"+testID+"/rotate", map[string]any{
		"newId": rotatedTestID, "salt": []byte("new-salt"), "ciphertext": []byte("new-cipher"), "newWriteToken": "new-token",
	}, "secret-token")
	if response.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d, want 409", response.Code)
	}
	if response := request(t, handler, http.MethodGet, "/api/pages/"+testID, nil, ""); response.Code != http.StatusOK {
		t.Fatalf("old page GET status = %d, want 200", response.Code)
	}
}

func TestFrontendSPAAndDevelopmentFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("SPA INDEX"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := newTestServer(t, os.DirFS(dir)).Handler()
	for _, path := range []string{"/", "/p/" + testID, "/unknown-client-route"} {
		response := request(t, handler, http.MethodGet, path, nil, "")
		if response.Code != http.StatusOK || response.Body.String() != "SPA INDEX" {
			t.Fatalf("%s: status=%d body=%q", path, response.Code, response.Body.String())
		}
	}

	fallbackHandler := newTestServer(t, nil).Handler()
	response := request(t, fallbackHandler, http.MethodGet, "/", nil, "")
	if response.Code != http.StatusServiceUnavailable || response.Body.String() != "fallback" {
		t.Fatalf("fallback: status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestInvalidInputAndMissingPage(t *testing.T) {
	handler := newTestServer(t, nil).Handler()
	if response := request(t, handler, http.MethodGet, "/api/pages/too-short", nil, ""); response.Code != http.StatusNotFound {
		t.Fatalf("invalid id status = %d", response.Code)
	}
	if response := request(t, handler, http.MethodGet, "/api/pages/0123456789abcdef", nil, ""); response.Code != http.StatusNotFound {
		t.Fatalf("missing page status = %d", response.Code)
	}
	response := request(t, handler, http.MethodPost, "/api/pages", map[string]any{"id": testID, "unknown": true}, "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown JSON field status = %d", response.Code)
	}
}

func TestCreationResourceLimits(t *testing.T) {
	t.Run("ciphertext size", func(t *testing.T) {
		config := DefaultConfig()
		config.MaxCiphertextBytes = 4
		config.CreateRatePerSecond = 0
		handler := newTestServerWithConfig(t, nil, config).Handler()
		response := request(t, handler, http.MethodPost, "/api/pages", map[string]any{
			"id": testID, "salt": []byte("salt"), "ciphertext": []byte("12345"), "writeToken": "token",
		}, "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("oversized ciphertext status = %d, want 400", response.Code)
		}
	})

	t.Run("page count", func(t *testing.T) {
		config := DefaultConfig()
		config.MaxPages = 1
		config.CreateRatePerSecond = 0
		handler := newTestServerWithConfig(t, nil, config).Handler()
		createPage(t, handler)
		response := request(t, handler, http.MethodPost, "/api/pages", map[string]any{
			"id": rotatedTestID, "salt": []byte("salt"), "ciphertext": []byte("cipher"), "writeToken": "token",
		}, "")
		if response.Code != statusInsufficientStorage {
			t.Fatalf("page quota status = %d, want %d", response.Code, statusInsufficientStorage)
		}
	})

	t.Run("database size", func(t *testing.T) {
		config := DefaultConfig()
		config.MaxRequestBodyBytes = 512 << 10
		config.MaxCiphertextBytes = 256 << 10
		config.MaxDatabaseBytes = 64 << 10
		config.CreateRatePerSecond = 0
		handler := newTestServerWithConfig(t, nil, config).Handler()
		response := request(t, handler, http.MethodPost, "/api/pages", map[string]any{
			"id": testID, "salt": []byte("salt"), "ciphertext": bytes.Repeat([]byte("x"), 128<<10), "writeToken": "token",
		}, "")
		if response.Code != statusInsufficientStorage {
			t.Fatalf("database quota status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	t.Run("create rate", func(t *testing.T) {
		config := DefaultConfig()
		config.CreateRatePerSecond = 0.000001
		config.CreateBurst = 1
		handler := newTestServerWithConfig(t, nil, config).Handler()
		createPage(t, handler)
		response := request(t, handler, http.MethodPost, "/api/pages", map[string]any{
			"id": rotatedTestID, "salt": []byte("salt"), "ciphertext": []byte("cipher"), "writeToken": "token",
		}, "")
		if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") == "" {
			t.Fatalf("rate limit status = %d retry-after = %q", response.Code, response.Header().Get("Retry-After"))
		}
	})
}

func TestDatabaseSizeLimitIsApplied(t *testing.T) {
	config := DefaultConfig()
	config.MaxDatabaseBytes = 8 << 20
	server := newTestServerWithConfig(t, nil, config)
	var pageSize, maxPageCount int64
	if err := server.db.QueryRow(`PRAGMA page_size`).Scan(&pageSize); err != nil {
		t.Fatal(err)
	}
	if err := server.db.QueryRow(`PRAGMA max_page_count`).Scan(&maxPageCount); err != nil {
		t.Fatal(err)
	}
	if got := maxPageCount * pageSize; got > config.MaxDatabaseBytes {
		t.Fatalf("database limit = %d, configured = %d", got, config.MaxDatabaseBytes)
	}
}

func TestAdminDeletionCapability(t *testing.T) {
	config := DefaultConfig()
	config.AdminToken = "admin-secret-with-at-least-32-bytes"
	handler := newTestServerWithConfig(t, nil, config).Handler()
	createPage(t, handler)

	path := "/api/admin/pages/" + testID
	if response := request(t, handler, http.MethodDelete, path, nil, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("missing admin token status = %d, want 401", response.Code)
	}
	if response := request(t, handler, http.MethodDelete, path, nil, "wrong"); response.Code != http.StatusForbidden {
		t.Fatalf("wrong admin token status = %d, want 403", response.Code)
	}
	if response := request(t, handler, http.MethodDelete, path, nil, config.AdminToken); response.Code != http.StatusNoContent {
		t.Fatalf("admin delete status = %d, body = %s", response.Code, response.Body.String())
	}
	if response := request(t, handler, http.MethodGet, "/api/pages/"+testID, nil, ""); response.Code != http.StatusNotFound {
		t.Fatalf("deleted page status = %d, want 404", response.Code)
	}

	disabled := newTestServer(t, nil).Handler()
	if response := request(t, disabled, http.MethodDelete, path, nil, config.AdminToken); response.Code != http.StatusNotFound {
		t.Fatalf("disabled admin API status = %d, want 404", response.Code)
	}
}

func TestBrowserSecurityHeaders(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("SPA INDEX"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := newTestServer(t, os.DirFS(dir)).Handler()
	req := httptest.NewRequest(http.MethodGet, "/p/"+testID, nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	for name, want := range map[string]string{
		"Content-Security-Policy":   "frame-ancestors 'none'",
		"X-Frame-Options":           "DENY",
		"Strict-Transport-Security": "max-age=31536000",
		"X-Robots-Tag":              "noindex",
	} {
		if got := response.Header().Get(name); !bytes.Contains([]byte(got), []byte(want)) {
			t.Errorf("%s = %q, want to contain %q", name, got, want)
		}
	}
}

func TestTrustedProxyUsesLastForwardedAddress(t *testing.T) {
	config := DefaultConfig()
	config.TrustProxyHeaders = true
	server := newTestServerWithConfig(t, nil, config)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.18.0.3:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.99, 198.51.100.7")
	if got := server.clientAddress(req); got != "198.51.100.7" {
		t.Fatalf("client address = %q, want last forwarded address", got)
	}

	server.config.TrustProxyHeaders = false
	if got := server.clientAddress(req); got != "172.18.0.3" {
		t.Fatalf("untrusted proxy client address = %q, want remote address", got)
	}
}
