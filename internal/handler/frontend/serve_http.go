package frontend

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

const indexFile = "index.html"

// ServeHTTP serves static assets and falls back to the SPA entry point.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	requestedPath := cleanPath(r.URL.Path)
	if requestedPath == "" || strings.HasPrefix(requestedPath, "p/") {
		requestedPath = indexFile
	}

	if h.serveFile(w, r, requestedPath) {
		return
	}

	if h.serveFile(w, r, indexFile) {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)

	_, err := io.Copy(w, bytes.NewReader(h.fallback))
	if err != nil {
		return
	}
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, name string) bool {
	if h.files == nil {
		return false
	}

	data, err := fs.ReadFile(h.files, name)
	if err != nil {
		return false
	}

	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))

	return true
}

func cleanPath(requestPath string) string {
	cleaned := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
	if cleaned == "." || !fs.ValidPath(cleaned) {
		return ""
	}

	return cleaned
}
