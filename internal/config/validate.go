package config

import (
	"fmt"
	"math"
	"net"
	"net/url"
	"slices"
	"strings"
)

func validate(loaded Config) error {
	err := validateHTTP(loaded.HTTP)
	if err != nil {
		return err
	}

	err = validateMetrics(loaded.HTTP, loaded.Metrics)
	if err != nil {
		return err
	}

	err = validateStorage(loaded)
	if err != nil {
		return err
	}

	err = validateProtection(loaded.Protection)
	if err != nil {
		return err
	}

	return validateCORS(loaded.CORS)
}

func validateMetrics(httpConfig HTTP, metricsConfig Metrics) error {
	metricsHost, metricsPort, err := net.SplitHostPort(metricsConfig.Listen)
	if err != nil {
		return fmt.Errorf("%w: %s must be a TCP listen address: %w",
			errInvalidEnvironment, envMetricsListen, err)
	}

	metricsIP := net.ParseIP(metricsHost)
	if metricsHost != "localhost" && (metricsIP == nil || !metricsIP.IsLoopback()) {
		return invalid(envMetricsListen, "must use a loopback address")
	}

	_, httpPort, err := net.SplitHostPort(httpConfig.Listen)
	if err != nil {
		return fmt.Errorf("%w: %s must be a TCP listen address: %w",
			errInvalidEnvironment, envHTTPListen, err)
	}

	if metricsPort == httpPort {
		return invalid(envMetricsListen, "must use a port different from the application server")
	}

	return nil
}

func validateHTTP(httpConfig HTTP) error {
	if strings.TrimSpace(httpConfig.Listen) == "" {
		return invalid(envHTTPListen, "must not be empty")
	}

	if httpConfig.ReadHeaderTimeout <= 0 || httpConfig.ReadTimeout <= 0 {
		return invalid("HTTP read timeouts", "must be positive")
	}

	if httpConfig.WriteTimeout <= 0 || httpConfig.IdleTimeout <= 0 {
		return invalid("HTTP write and idle timeouts", "must be positive")
	}

	if httpConfig.ShutdownTimeout <= 0 {
		return invalid(envHTTPShutdownTimeout, "must be positive")
	}

	return nil
}

func validateStorage(loaded Config) error {
	if strings.TrimSpace(loaded.SQLite.DSN) == "" {
		return invalid(envSQLiteDSN, "must not be empty")
	}

	if loaded.SQLite.MaxBytes < 0 || loaded.Page.MaxPages < 0 {
		return invalid("storage limits", "must not be negative")
	}

	if loaded.Protection.MaxRequestBodyBytes < 1 {
		return invalid(envRequestMaxBody, "must be positive")
	}

	return nil
}

func validateProtection(protection Protection) error {
	err := validateRate(protection.CreateRatePerSecond, protection.CreateBurst,
		envCreateRate, envCreateBurst)
	if err != nil {
		return err
	}

	err = validateRate(protection.AdminRatePerSecond, protection.AdminBurst,
		envAdminRate, envAdminBurst)
	if err != nil {
		return err
	}

	if protection.AdminToken == "" {
		return nil
	}

	if len(protection.AdminToken) < minAdminTokenBytes || len(protection.AdminToken) > maxAdminTokenBytes {
		return invalid(envAdminTokenFile, "token must contain between 32 and 256 bytes")
	}

	return nil
}

func validateRate(rate float64, burst int, rateName, burstName string) error {
	if rate < 0 || math.IsNaN(rate) {
		return invalid(rateName, "must be a non-negative finite number")
	}

	if math.IsInf(rate, 0) {
		return invalid(rateName, "must be a non-negative finite number")
	}

	if burst < 0 {
		return invalid(burstName, "must not be negative")
	}

	if rate > 0 && burst < 1 {
		return invalid(burstName, "must be positive when rate limiting is enabled")
	}

	return nil
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

func invalid(name, message string) error {
	return fmt.Errorf("%w: %s %s", errInvalidEnvironment, name, message)
}
