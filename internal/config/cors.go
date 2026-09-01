package config

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	envCORSAllowedOrigins   = "SINGLEPAGE_CORS_ALLOWED_ORIGINS"
	envCORSAllowedMethods   = "SINGLEPAGE_CORS_ALLOWED_METHODS"
	envCORSAllowedHeaders   = "SINGLEPAGE_CORS_ALLOWED_HEADERS"
	envCORSExposedHeaders   = "SINGLEPAGE_CORS_EXPOSED_HEADERS"
	envCORSAllowCredentials = "SINGLEPAGE_CORS_ALLOW_CREDENTIALS" //nolint:gosec // Environment name.
	envCORSMaxAge           = "SINGLEPAGE_CORS_MAX_AGE"
)

// CORS contains the explicit cross-origin resource sharing policy.
type CORS struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

func defaultCORS() CORS {
	return CORS{
		AllowedOrigins:   []string{},
		AllowedMethods:   []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           0,
	}
}

func loadCORS(corsConfig *CORS) error {
	corsConfig.AllowedOrigins = parseList(envCORSAllowedOrigins, nil)
	corsConfig.AllowedMethods = parseList(envCORSAllowedMethods, corsConfig.AllowedMethods)
	corsConfig.AllowedHeaders = parseList(envCORSAllowedHeaders, corsConfig.AllowedHeaders)
	corsConfig.ExposedHeaders = parseList(envCORSExposedHeaders, corsConfig.ExposedHeaders)

	err := parseBool(&corsConfig.AllowCredentials, envCORSAllowCredentials)
	if err != nil {
		return err
	}

	return parseDuration(&corsConfig.MaxAge, envCORSMaxAge)
}

func validateCORS(corsConfig CORS) error {
	if corsConfig.MaxAge < 0 {
		return invalid(envCORSMaxAge, "must not be negative")
	}

	if corsConfig.AllowCredentials && slices.Contains(corsConfig.AllowedOrigins, "*") {
		return invalid(envCORSAllowedOrigins, "wildcard cannot be used with credentials")
	}

	for _, origin := range corsConfig.AllowedOrigins {
		err := validateOrigin(origin)
		if err != nil {
			return err
		}
	}

	for _, method := range corsConfig.AllowedMethods {
		if !validMethod(method) {
			return invalid(envCORSAllowedMethods, "contains an invalid HTTP method")
		}
	}

	headers := append(clone(corsConfig.AllowedHeaders), corsConfig.ExposedHeaders...)

	for _, header := range headers {
		if !validHeader(header) {
			return invalid("CORS headers", "contains an invalid HTTP header name")
		}
	}

	return nil
}

func validateOrigin(origin string) error {
	if origin == "*" {
		return nil
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("%w: %s contains an invalid origin: %w", errInvalidEnvironment,
			envCORSAllowedOrigins, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return invalid(envCORSAllowedOrigins, "origins must use http or https")
	}

	if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return invalid(envCORSAllowedOrigins, "contains an invalid origin")
	}

	if parsed.Path != "" {
		return invalid(envCORSAllowedOrigins, "origins must not contain a path")
	}

	return nil
}

func validMethod(method string) bool {
	return strings.ToUpper(method) == method && validToken(method)
}

func validHeader(header string) bool {
	return header == "*" || validToken(header)
}

func validToken(value string) bool {
	if value == "" {
		return false
	}

	for index := range len(value) {
		if !isTokenCharacter(value[index]) {
			return false
		}
	}

	return true
}

func isTokenCharacter(character byte) bool {
	if character >= '0' && character <= '9' {
		return true
	}

	if character >= 'A' && character <= 'Z' {
		return true
	}

	if character >= 'a' && character <= 'z' {
		return true
	}

	return strings.ContainsRune("!#$%&'*+-.^_`|~", rune(character))
}

func clone(values []string) []string {
	return append([]string(nil), values...)
}
