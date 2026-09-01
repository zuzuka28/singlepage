package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const maxRequestBody = 16 << 20 // 16 MiB of JSON (ciphertext is base64 encoded).

// Server stores encrypted page objects. It deliberately has no knowledge of
// the decrypted document or any outline concepts.
type Server struct {
	db       *sql.DB
	frontend fs.FS
	fallback []byte
}

type pageResponse struct {
	Revision   int64  `json:"revision"`
	Salt       []byte `json:"salt"`
	Ciphertext []byte `json:"ciphertext"`
}

type createRequest struct {
	ID         string `json:"id"`
	Salt       []byte `json:"salt"`
	Ciphertext []byte `json:"ciphertext"`
	WriteToken string `json:"writeToken"`
}

type updateRequest struct {
	ExpectedRevision int64   `json:"expectedRevision"`
	Ciphertext       []byte  `json:"ciphertext"`
	Salt             *[]byte `json:"salt,omitempty"`
	NewWriteToken    string  `json:"newWriteToken,omitempty"`
	// WriteToken is accepted for clients that cannot set Authorization. New
	// clients should use an Authorization: Bearer header instead.
	WriteToken string `json:"writeToken,omitempty"`
}

type rotateRequest struct {
	NewID         string `json:"newId"`
	Salt          []byte `json:"salt"`
	Ciphertext    []byte `json:"ciphertext"`
	NewWriteToken string `json:"newWriteToken"`
}

func Open(ctx context.Context, dsn string, frontend fs.FS, fallback []byte) (*Server, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// A small single-binary service benefits from deterministic SQLite locking,
	// and one writer connection is sufficient for its CAS workload.
	db.SetMaxOpenConns(1)
	if _, err = db.ExecContext(ctx, `
		PRAGMA journal_mode=WAL;
		PRAGMA busy_timeout=5000;
		CREATE TABLE IF NOT EXISTS pages (
			id TEXT PRIMARY KEY,
			revision INTEGER NOT NULL CHECK (revision >= 1),
			salt BLOB NOT NULL,
			write_token_hash BLOB NOT NULL CHECK (length(write_token_hash) = 32),
			ciphertext BLOB NOT NULL,
			updated_at INTEGER NOT NULL
		);`); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize sqlite: %w", err)
	}
	return &Server{db: db, frontend: frontend, fallback: fallback}, nil
}

func (s *Server) Close() error { return s.db.Close() }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/pages", s.handlePages)
	mux.HandleFunc("/api/pages/", s.handlePage)
	mux.HandleFunc("/", s.handleFrontend)
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handlePages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var req createRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateID(req.ID); err != nil || len(req.Salt) == 0 || len(req.Ciphertext) == 0 || req.WriteToken == "" {
		writeError(w, http.StatusBadRequest, "id, salt, ciphertext, and writeToken are required")
		return
	}
	hash := sha256.Sum256([]byte(req.WriteToken))
	_, err := s.db.ExecContext(r.Context(), `
		INSERT INTO pages (id, revision, salt, write_token_hash, ciphertext, updated_at)
		VALUES (?, 1, ?, ?, ?, ?)`, req.ID, req.Salt, hash[:], req.Ciphertext, time.Now().Unix())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(w, http.StatusConflict, "page already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to create page")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"revision": 1})
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/pages/")
	id, action, hasAction := strings.Cut(path, "/")
	if validateID(id) != nil || (hasAction && action != "rotate") || strings.Contains(action, "/") {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	if hasAction {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		s.rotatePage(w, r, id)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getPage(w, r, id)
	case http.MethodPut:
		s.updatePage(w, r, id)
	default:
		methodNotAllowed(w, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) rotatePage(w http.ResponseWriter, r *http.Request, oldID string) {
	var req rotateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if validateID(req.NewID) != nil || req.NewID == oldID || len(req.Salt) == 0 || len(req.Ciphertext) == 0 || req.NewWriteToken == "" {
		writeError(w, http.StatusBadRequest, "newId, salt, ciphertext, and newWriteToken are required")
		return
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "write authorization required")
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to rotate page")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var storedHash []byte
	err = tx.QueryRowContext(r.Context(),
		`SELECT write_token_hash FROM pages WHERE id = ?`, oldID,
	).Scan(&storedHash)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to rotate page")
		return
	}
	providedHash := sha256.Sum256([]byte(token))
	if len(storedHash) != len(providedHash) || subtle.ConstantTimeCompare(storedHash, providedHash[:]) != 1 {
		writeError(w, http.StatusForbidden, "invalid write authorization")
		return
	}

	newHash := sha256.Sum256([]byte(req.NewWriteToken))
	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO pages (id, revision, salt, write_token_hash, ciphertext, updated_at)
		VALUES (?, 1, ?, ?, ?, ?)`, req.NewID, req.Salt, newHash[:], req.Ciphertext, time.Now().Unix())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(w, http.StatusConflict, "page already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to rotate page")
		return
	}
	result, err := tx.ExecContext(r.Context(), `DELETE FROM pages WHERE id = ? AND write_token_hash = ?`, oldID, storedHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to rotate page")
		return
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		writeError(w, http.StatusConflict, "page changed during rotation")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to rotate page")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"revision": 1})
}

func (s *Server) getPage(w http.ResponseWriter, r *http.Request, id string) {
	var page pageResponse
	err := s.db.QueryRowContext(r.Context(),
		`SELECT revision, salt, ciphertext FROM pages WHERE id = ?`, id,
	).Scan(&page.Revision, &page.Salt, &page.Ciphertext)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to read page")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) updatePage(w http.ResponseWriter, r *http.Request, id string) {
	var req updateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ExpectedRevision < 1 || len(req.Ciphertext) == 0 {
		writeError(w, http.StatusBadRequest, "expectedRevision and ciphertext are required")
		return
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		token = req.WriteToken
	}
	if token == "" {
		writeError(w, http.StatusUnauthorized, "write authorization required")
		return
	}

	var currentRevision int64
	var currentSalt, storedHash []byte
	err := s.db.QueryRowContext(r.Context(),
		`SELECT revision, salt, write_token_hash FROM pages WHERE id = ?`, id,
	).Scan(&currentRevision, &currentSalt, &storedHash)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to update page")
		return
	}
	providedHash := sha256.Sum256([]byte(token))
	if len(storedHash) != len(providedHash) || subtle.ConstantTimeCompare(storedHash, providedHash[:]) != 1 {
		writeError(w, http.StatusForbidden, "invalid write authorization")
		return
	}
	if currentRevision != req.ExpectedRevision {
		writeError(w, http.StatusConflict, "revision conflict")
		return
	}

	salt := currentSalt
	if req.Salt != nil {
		if len(*req.Salt) == 0 {
			writeError(w, http.StatusBadRequest, "salt cannot be empty")
			return
		}
		salt = *req.Salt
	}
	newHash := storedHash
	if req.NewWriteToken != "" {
		h := sha256.Sum256([]byte(req.NewWriteToken))
		newHash = h[:]
	}
	result, err := s.db.ExecContext(r.Context(), `
		UPDATE pages
		SET revision = revision + 1, salt = ?, write_token_hash = ?, ciphertext = ?, updated_at = ?
		WHERE id = ? AND revision = ? AND write_token_hash = ?`,
		salt, newHash, req.Ciphertext, time.Now().Unix(), id, req.ExpectedRevision, storedHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to update page")
		return
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		writeError(w, http.StatusConflict, "revision conflict")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"revision": req.ExpectedRevision + 1})
}

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet+", "+http.MethodHead)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || strings.HasPrefix(path, "p/") {
		path = "index.html"
	}
	if s.frontend != nil {
		if data, err := fs.ReadFile(s.frontend, path); err == nil {
			if contentType := mime.TypeByExtension(filepathExtension(path)); contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
			_, _ = w.Write(data)
			return
		}
		// Client-side routes must serve the SPA entry point.
		if data, err := fs.ReadFile(s.frontend, "index.html"); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(data)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write(s.fallback)
}

func filepathExtension(path string) string {
	if slash := strings.LastIndexByte(path, '/'); slash >= 0 {
		path = path[slash+1:]
	}
	if dot := strings.LastIndexByte(path, '.'); dot >= 0 {
		return path[dot:]
	}
	return ""
}

func validateID(id string) error {
	if len(id) < 16 || len(id) > 128 {
		return errors.New("invalid id")
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return errors.New("invalid id")
		}
	}
	return nil
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid JSON body")
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("invalid JSON body")
	}
	return nil
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
