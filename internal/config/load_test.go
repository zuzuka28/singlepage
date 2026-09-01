package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"singlepage/internal/config"
)

const (
	envReadTimeout     = "SINGLEPAGE_HTTP_READ_TIMEOUT"
	envMetricsListen   = "SINGLEPAGE_METRICS_LISTEN"
	envSQLiteMaxBytes  = "SINGLEPAGE_SQLITE_MAX_BYTES"
	envCreateRate      = "SINGLEPAGE_CREATE_RATE_PER_SECOND"
	envAdminRate       = "SINGLEPAGE_ADMIN_RATE_PER_SECOND"
	envAdminTokenFile  = "SINGLEPAGE_ADMIN_TOKEN_FILE"
	envCORSOrigins     = "SINGLEPAGE_CORS_ALLOWED_ORIGINS"
	envCORSMethods     = "SINGLEPAGE_CORS_ALLOWED_METHODS"
	envCORSCredentials = "SINGLEPAGE_CORS_ALLOW_CREDENTIALS" //nolint:gosec // Environment name.
	requestIDHeader    = "X-Request-ID"
	enabled            = "true"
)

//nolint:paralleltest // Tests mutate process environment through t.Setenv.
func TestLoadDefaults(t *testing.T) {
	clearEnvironment(t)

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := config.Config{
		HTTP: config.HTTP{
			Listen:            "127.0.0.1:8080",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       20 * time.Second,
			WriteTimeout:      20 * time.Second,
			IdleTimeout:       time.Minute,
			ShutdownTimeout:   10 * time.Second,
		},
		Metrics: config.Metrics{Listen: "127.0.0.1:9090"},
		SQLite:  config.SQLite{DSN: "data.db", MaxBytes: 512 << 20},
		Page:    config.Page{MaxPages: 100_000},
		Protection: config.Protection{
			MaxRequestBodyBytes: 16 << 20,
			CreateRatePerSecond: 1,
			CreateBurst:         20,
			AdminRatePerSecond:  1,
			AdminBurst:          5,
		},
		CORS: config.CORS{
			AllowedMethods: []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Authorization", "Content-Type"},
			ExposedHeaders: []string{requestIDHeader},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

//nolint:paralleltest // Tests mutate process environment through t.Setenv.
func TestLoadEnvironment(t *testing.T) {
	clearEnvironment(t)
	tokenPath := filepath.Join(t.TempDir(), "admin-token")

	token := strings.Repeat("s", 32)

	err := os.WriteFile(tokenPath, []byte("  "+token+"\n"), 0o600)
	if err != nil {
		t.Fatalf("write token: %v", err)
	}

	values := map[string]string{
		"SINGLEPAGE_HTTP_LISTEN":              ":9090",
		"SINGLEPAGE_HTTP_READ_HEADER_TIMEOUT": "1s",
		envReadTimeout:                        "2s",
		"SINGLEPAGE_HTTP_WRITE_TIMEOUT":       "3s",
		"SINGLEPAGE_HTTP_IDLE_TIMEOUT":        "4s",
		"SINGLEPAGE_HTTP_SHUTDOWN_TIMEOUT":    "5s",
		envMetricsListen:                      "localhost:9191",
		"SINGLEPAGE_SQLITE_DSN":               "file:test.db?_journal_mode=WAL",
		envSQLiteMaxBytes:                     "1000",
		"SINGLEPAGE_PAGE_MAX_PAGES":           "30",
		"SINGLEPAGE_REQUEST_MAX_BODY_BYTES":   "3000",
		envCreateRate:                         "2.5",
		"SINGLEPAGE_CREATE_BURST":             "7",
		envAdminRate:                          "3.5",
		"SINGLEPAGE_ADMIN_BURST":              "9",
		"SINGLEPAGE_TRUST_PROXY_HEADERS":      enabled,
		envAdminTokenFile:                     tokenPath,
		envCORSOrigins:                        "https://one.example, https://two.example",
		envCORSMethods:                        "GET, POST",
		"SINGLEPAGE_CORS_ALLOWED_HEADERS":     "Authorization, " + requestIDHeader,
		"SINGLEPAGE_CORS_EXPOSED_HEADERS":     "ETag, " + requestIDHeader,
		envCORSCredentials:                    enabled,
		"SINGLEPAGE_CORS_MAX_AGE":             "10m",
	}
	for name, value := range values {
		t.Setenv(name, value)
	}

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertEnvironmentConfig(t, got, token)
}

func assertEnvironmentConfig(t *testing.T, got config.Config, token string) {
	t.Helper()

	want := config.Config{
		HTTP: config.HTTP{
			Listen:            ":9090",
			ReadHeaderTimeout: time.Second,
			ReadTimeout:       2 * time.Second,
			WriteTimeout:      3 * time.Second,
			IdleTimeout:       4 * time.Second,
			ShutdownTimeout:   5 * time.Second,
		},
		Metrics: config.Metrics{Listen: "localhost:9191"},
		SQLite:  config.SQLite{DSN: "file:test.db?_journal_mode=WAL", MaxBytes: 1000},
		Page:    config.Page{MaxPages: 30},
		Protection: config.Protection{
			MaxRequestBodyBytes: 3000,
			CreateRatePerSecond: 2.5,
			CreateBurst:         7,
			AdminRatePerSecond:  3.5,
			AdminBurst:          9,
			TrustProxyHeaders:   true,
			AdminToken:          token,
		},
		CORS: config.CORS{
			AllowedOrigins:   []string{"https://one.example", "https://two.example"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Authorization", requestIDHeader},
			ExposedHeaders:   []string{"ETag", requestIDHeader},
			AllowCredentials: true,
			MaxAge:           10 * time.Minute,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	emptyTokenPath := filepath.Join(t.TempDir(), "empty-token")

	err := os.WriteFile(emptyTokenPath, []byte(" \n"), 0o600)
	if err != nil {
		t.Fatalf("write empty token: %v", err)
	}

	tests := []struct {
		name  string
		env   map[string]string
		match string
	}{
		{
			name:  "invalid duration",
			env:   map[string]string{envReadTimeout: "later"},
			match: envReadTimeout,
		},
		{
			name:  "public metrics address",
			env:   map[string]string{envMetricsListen: "0.0.0.0:9090"},
			match: "loopback",
		},
		{
			name:  "shared application and metrics port",
			env:   map[string]string{envMetricsListen: "127.0.0.1:8080"},
			match: "different",
		},
		{
			name:  "negative limit",
			env:   map[string]string{envSQLiteMaxBytes: "-1"},
			match: "storage limits",
		},
		{
			name: "wildcard origin with credentials",
			env: map[string]string{
				envCORSOrigins:     "*",
				envCORSCredentials: enabled,
			},
			match: "wildcard",
		},
		{
			name:  "origin path",
			env:   map[string]string{envCORSOrigins: "https://example.com/path"},
			match: "path",
		},
		{
			name:  "invalid method",
			env:   map[string]string{envCORSMethods: "get"},
			match: "HTTP method",
		},
		{
			name:  "infinite rate",
			env:   map[string]string{envCreateRate: "+Inf"},
			match: "finite",
		},
		{
			name:  "missing token file",
			env:   map[string]string{envAdminTokenFile: "/does/not/exist"},
			match: envAdminTokenFile,
		},
		{
			name:  "empty token file",
			env:   map[string]string{envAdminTokenFile: emptyTokenPath},
			match: "must not be empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearEnvironment(t)

			for name, value := range test.env {
				t.Setenv(name, value)
			}

			_, err := config.Load()
			if err == nil || !strings.Contains(err.Error(), test.match) {
				t.Fatalf("Load() error = %v, want containing %q", err, test.match)
			}
		})
	}
}

func clearEnvironment(t *testing.T) {
	t.Helper()

	for _, name := range environmentNames() {
		t.Setenv(name, "")
	}
}

func environmentNames() []string {
	return []string{
		"SINGLEPAGE_HTTP_LISTEN",
		"SINGLEPAGE_HTTP_READ_HEADER_TIMEOUT",
		envReadTimeout,
		"SINGLEPAGE_HTTP_WRITE_TIMEOUT",
		"SINGLEPAGE_HTTP_IDLE_TIMEOUT",
		"SINGLEPAGE_HTTP_SHUTDOWN_TIMEOUT",
		envMetricsListen,
		"SINGLEPAGE_SQLITE_DSN",
		envSQLiteMaxBytes,
		"SINGLEPAGE_PAGE_MAX_PAGES",
		"SINGLEPAGE_REQUEST_MAX_BODY_BYTES",
		envCreateRate,
		"SINGLEPAGE_CREATE_BURST",
		envAdminRate,
		"SINGLEPAGE_ADMIN_BURST",
		"SINGLEPAGE_TRUST_PROXY_HEADERS",
		envAdminTokenFile,
		envCORSOrigins,
		envCORSMethods,
		"SINGLEPAGE_CORS_ALLOWED_HEADERS",
		"SINGLEPAGE_CORS_EXPOSED_HEADERS",
		envCORSCredentials,
		"SINGLEPAGE_CORS_MAX_AGE",
	}
}
