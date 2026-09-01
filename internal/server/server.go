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
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
)

const (
	defaultMaxRequestBodyBytes = 3 << 20   // JSON includes base64-encoded ciphertext.
	defaultMaxCiphertextBytes  = 2 << 20   // 2 MiB encrypted page payload.
	defaultMaxDatabaseBytes    = 512 << 20 // SQLite logical page limit.
	defaultMaxPages            = 100_000
	defaultCreateRatePerSecond = 1
	defaultCreateBurst         = 20
	maxRateLimitClients        = 10_000
	maxSaltBytes               = 64
	maxCapabilityBytes         = 256
	minAdminTokenBytes         = 32
	statusInsufficientStorage  = 507
)

// Config bounds the resources exposed by the unauthenticated create API.
// Zero values disable the corresponding database, page-count, or rate limit.
type Config struct {
	MaxRequestBodyBytes int64
	MaxCiphertextBytes  int
	MaxDatabaseBytes    int64
	MaxPages            int64
	CreateRatePerSecond float64
	CreateBurst         int
	TrustProxyHeaders   bool
	AdminToken          string
}

func DefaultConfig() Config {
	return Config{
		MaxRequestBodyBytes: defaultMaxRequestBodyBytes,
		MaxCiphertextBytes:  defaultMaxCiphertextBytes,
		MaxDatabaseBytes:    defaultMaxDatabaseBytes,
		MaxPages:            defaultMaxPages,
		CreateRatePerSecond: defaultCreateRatePerSecond,
		CreateBurst:         defaultCreateBurst,
	}
}

// Server stores encrypted page objects. It deliberately has no knowledge of
// the decrypted document or any outline concepts.
type Server struct {
	db             *sql.DB
	frontend       fs.FS
	fallback       []byte
	config         Config
	createLimiter  *clientRateLimiter
	adminLimiter   *clientRateLimiter
	adminTokenHash []byte
}

type tokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

type clientRateLimiter struct {
	mu        sync.Mutex
	rate      float64
	burst     float64
	buckets   map[string]*tokenBucket
	lastSeen  map[string]time.Time
	lastSweep time.Time
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
	return OpenWithConfig(ctx, dsn, frontend, fallback, DefaultConfig())
}

func OpenWithConfig(ctx context.Context, dsn string, frontend fs.FS, fallback []byte, config Config) (*Server, error) {
	if config.MaxRequestBodyBytes < 1 || config.MaxCiphertextBytes < 1 {
		return nil, errors.New("request and ciphertext limits must be positive")
	}
	if config.MaxDatabaseBytes < 0 || config.MaxPages < 0 || config.CreateRatePerSecond < 0 || config.CreateBurst < 0 {
		return nil, errors.New("resource limits cannot be negative")
	}
	if config.CreateRatePerSecond > 0 && config.CreateBurst < 1 {
		return nil, errors.New("create burst must be positive when rate limiting is enabled")
	}
	if config.AdminToken != "" && (len(config.AdminToken) < minAdminTokenBytes || len(config.AdminToken) > maxCapabilityBytes) {
		return nil, fmt.Errorf("admin token must contain between %d and %d bytes", minAdminTokenBytes, maxCapabilityBytes)
	}

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
	if config.MaxDatabaseBytes > 0 {
		var pageSize, pageCount int64
		if err = db.QueryRowContext(ctx, `PRAGMA page_size`).Scan(&pageSize); err != nil {
			db.Close()
			return nil, fmt.Errorf("read sqlite page size: %w", err)
		}
		if err = db.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&pageCount); err != nil {
			db.Close()
			return nil, fmt.Errorf("read sqlite page count: %w", err)
		}
		maxPageCount := config.MaxDatabaseBytes / pageSize
		if maxPageCount < pageCount {
			db.Close()
			return nil, fmt.Errorf("sqlite database already exceeds configured storage limit")
		}
		var applied int64
		if err = db.QueryRowContext(ctx, fmt.Sprintf(`PRAGMA max_page_count = %d`, maxPageCount)).Scan(&applied); err != nil {
			db.Close()
			return nil, fmt.Errorf("limit sqlite database size: %w", err)
		}
		if applied > maxPageCount {
			db.Close()
			return nil, errors.New("sqlite refused configured storage limit")
		}
	}

	adminToken := config.AdminToken
	config.AdminToken = ""
	server := &Server{db: db, frontend: frontend, fallback: fallback, config: config}
	if config.CreateRatePerSecond > 0 {
		server.createLimiter = newClientRateLimiter(config.CreateRatePerSecond, config.CreateBurst)
	}
	if adminToken != "" {
		hash := sha256.Sum256([]byte(adminToken))
		server.adminTokenHash = hash[:]
		server.adminLimiter = newClientRateLimiter(1, 5)
	}
	return server, nil
}

func (s *Server) Close() error { return s.db.Close() }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/pages", s.handlePages)
	mux.HandleFunc("/api/pages/", s.handlePage)
	mux.HandleFunc("/api/admin/pages/", s.handleAdminPage)
	mux.HandleFunc("/", s.handleFrontend)
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=(), payment=(), usb=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data: blob:; font-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; manifest-src 'self'")
		w.Header().Set("Cache-Control", "no-store")
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		if strings.HasPrefix(r.URL.Path, "/p/") {
			w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handlePages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if s.createLimiter != nil && !s.createLimiter.Allow(s.clientAddress(r), time.Now()) {
		w.Header().Set("Retry-After", "1")
		writeError(w, http.StatusTooManyRequests, "page creation rate exceeded")
		return
	}
	var req createRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateID(req.ID); err != nil || !s.validSalt(req.Salt) || !s.validCiphertext(req.Ciphertext) || !validCapability(req.WriteToken) {
		writeError(w, http.StatusBadRequest, "id, salt, ciphertext, and writeToken are required")
		return
	}
	hash := sha256.Sum256([]byte(req.WriteToken))
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeDatabaseError(w, err, "unable to create page")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if s.config.MaxPages > 0 {
		var count int64
		if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM pages`).Scan(&count); err != nil {
			writeDatabaseError(w, err, "unable to create page")
			return
		}
		if count >= s.config.MaxPages {
			writeError(w, statusInsufficientStorage, "page storage quota exceeded")
			return
		}
	}
	_, err = tx.ExecContext(r.Context(), `
			INSERT INTO pages (id, revision, salt, write_token_hash, ciphertext, updated_at)
			VALUES (?, 1, ?, ?, ?, ?)`, req.ID, req.Salt, hash[:], req.Ciphertext, time.Now().Unix())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(w, http.StatusConflict, "page already exists")
			return
		}
		writeDatabaseError(w, err, "unable to create page")
		return
	}
	if err = tx.Commit(); err != nil {
		writeDatabaseError(w, err, "unable to create page")
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
	if err := s.decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if validateID(req.NewID) != nil || req.NewID == oldID || !s.validSalt(req.Salt) || !s.validCiphertext(req.Ciphertext) || !validCapability(req.NewWriteToken) {
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
		writeDatabaseError(w, err, "unable to rotate page")
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
	result, err := tx.ExecContext(r.Context(), `
		UPDATE pages
		SET id = ?, revision = 1, salt = ?, write_token_hash = ?, ciphertext = ?, updated_at = ?
		WHERE id = ? AND write_token_hash = ?`,
		req.NewID, req.Salt, newHash[:], req.Ciphertext, time.Now().Unix(), oldID, storedHash)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeError(w, http.StatusConflict, "page already exists")
			return
		}
		writeDatabaseError(w, err, "unable to rotate page")
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
	if err := s.decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ExpectedRevision < 1 || !s.validCiphertext(req.Ciphertext) {
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
		if !s.validSalt(*req.Salt) {
			writeError(w, http.StatusBadRequest, "salt cannot be empty")
			return
		}
		salt = *req.Salt
	}
	newHash := storedHash
	if req.NewWriteToken != "" {
		if !validCapability(req.NewWriteToken) {
			writeError(w, http.StatusBadRequest, "new write token is invalid")
			return
		}
		h := sha256.Sum256([]byte(req.NewWriteToken))
		newHash = h[:]
	}
	result, err := s.db.ExecContext(r.Context(), `
		UPDATE pages
		SET revision = revision + 1, salt = ?, write_token_hash = ?, ciphertext = ?, updated_at = ?
		WHERE id = ? AND revision = ? AND write_token_hash = ?`,
		salt, newHash, req.Ciphertext, time.Now().Unix(), id, req.ExpectedRevision, storedHash)
	if err != nil {
		writeDatabaseError(w, err, "unable to update page")
		return
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		writeError(w, http.StatusConflict, "revision conflict")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"revision": req.ExpectedRevision + 1})
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	if len(s.adminTokenHash) == 0 {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}
	if !s.adminLimiter.Allow(s.clientAddress(r), time.Now()) {
		w.Header().Set("Retry-After", "1")
		writeError(w, http.StatusTooManyRequests, "admin authorization rate exceeded")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/pages/")
	if validateID(id) != nil || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "admin authorization required")
		return
	}
	providedHash := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(s.adminTokenHash, providedHash[:]) != 1 {
		writeError(w, http.StatusForbidden, "invalid admin authorization")
		return
	}
	result, err := s.db.ExecContext(r.Context(), `DELETE FROM pages WHERE id = ?`, id)
	if err != nil {
		writeDatabaseError(w, err, "unable to delete page")
		return
	}
	rows, err := result.RowsAffected()
	if err != nil {
		writeDatabaseError(w, err, "unable to delete page")
		return
	}
	if rows != 1 {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxRequestBodyBytes)
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

func (s *Server) validSalt(salt []byte) bool {
	return len(salt) > 0 && len(salt) <= maxSaltBytes
}

func (s *Server) validCiphertext(ciphertext []byte) bool {
	return len(ciphertext) > 0 && len(ciphertext) <= s.config.MaxCiphertextBytes
}

func validCapability(token string) bool {
	return token != "" && len(token) <= maxCapabilityBytes
}

func (l *tokenBucket) Allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	elapsed := now.Sub(l.last).Seconds()
	if elapsed > 0 {
		l.tokens = min(l.burst, l.tokens+elapsed*l.rate)
		l.last = now
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

func (l *clientRateLimiter) Allow(client string, now time.Time) bool {
	l.mu.Lock()
	if now.Sub(l.lastSweep) >= time.Minute {
		cutoff := now.Add(-10 * time.Minute)
		for key, seen := range l.lastSeen {
			if seen.Before(cutoff) {
				delete(l.lastSeen, key)
				delete(l.buckets, key)
			}
		}
		l.lastSweep = now
	}
	bucket := l.buckets[client]
	if bucket == nil {
		if len(l.buckets) >= maxRateLimitClients {
			l.mu.Unlock()
			return false
		}
		bucket = &tokenBucket{rate: l.rate, burst: l.burst, tokens: l.burst, last: now}
		l.buckets[client] = bucket
	}
	l.lastSeen[client] = now
	l.mu.Unlock()
	return bucket.Allow(now)
}

func newClientRateLimiter(rate float64, burst int) *clientRateLimiter {
	return &clientRateLimiter{
		rate: rate, burst: float64(burst), buckets: make(map[string]*tokenBucket),
		lastSeen: make(map[string]time.Time), lastSweep: time.Now(),
	}
}

func (s *Server) clientAddress(r *http.Request) string {
	address := r.RemoteAddr
	if s.config.TrustProxyHeaders {
		forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		if candidate := strings.TrimSpace(forwarded[len(forwarded)-1]); net.ParseIP(candidate) != nil {
			address = candidate
		}
	}
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}

func writeDatabaseError(w http.ResponseWriter, err error, fallback string) {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrFull {
		writeError(w, statusInsufficientStorage, "page storage quota exceeded")
		return
	}
	writeError(w, http.StatusInternalServerError, fallback)
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
