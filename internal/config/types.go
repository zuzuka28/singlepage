// Package config loads and validates the application's environment configuration.
package config

import "time"

// Config contains all runtime configuration.
type Config struct {
	HTTP       HTTP
	Metrics    Metrics
	SQLite     SQLite
	Page       Page
	Protection Protection
	CORS       CORS
}

// Metrics contains the private Prometheus server settings.
type Metrics struct {
	Listen string
}

// HTTP contains HTTP server lifecycle settings.
type HTTP struct {
	Listen            string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// SQLite contains SQLite storage settings.
type SQLite struct {
	DSN      string
	MaxBytes int64
}

// Page contains limits applied to encrypted pages.
type Page struct {
	MaxPages int64
}

// Protection contains request protection settings.
type Protection struct {
	MaxRequestBodyBytes int64
	CreateRatePerSecond float64
	CreateBurst         int
	AdminRatePerSecond  float64
	AdminBurst          int
	TrustProxyHeaders   bool
	AdminToken          string
}

// CORS contains the explicit cross-origin resource sharing policy.
type CORS struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}
