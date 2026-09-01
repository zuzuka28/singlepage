package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func loadDurations(loaded *Config) error {
	values := []struct {
		name   string
		target *time.Duration
	}{
		{name: envHTTPReadHeaderTimeout, target: &loaded.HTTP.ReadHeaderTimeout},
		{name: envHTTPReadTimeout, target: &loaded.HTTP.ReadTimeout},
		{name: envHTTPWriteTimeout, target: &loaded.HTTP.WriteTimeout},
		{name: envHTTPIdleTimeout, target: &loaded.HTTP.IdleTimeout},
		{name: envHTTPShutdownTimeout, target: &loaded.HTTP.ShutdownTimeout},
		{name: envCORSMaxAge, target: &loaded.CORS.MaxAge},
	}

	for _, value := range values {
		err := parseDuration(value.target, value.name)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadNumbers(loaded *Config) error {
	integers := []struct {
		name   string
		target *int64
	}{
		{name: envSQLiteMaxBytes, target: &loaded.SQLite.MaxBytes},
		{name: envPageMaxPages, target: &loaded.Page.MaxPages},
		{name: envRequestMaxBody, target: &loaded.Protection.MaxRequestBodyBytes},
	}

	for _, integer := range integers {
		err := parseInt64(integer.target, integer.name)
		if err != nil {
			return err
		}
	}

	err := parseInt(&loaded.Protection.CreateBurst, envCreateBurst)
	if err != nil {
		return err
	}

	err = parseInt(&loaded.Protection.AdminBurst, envAdminBurst)
	if err != nil {
		return err
	}

	err = parseFloat64(&loaded.Protection.CreateRatePerSecond, envCreateRate)
	if err != nil {
		return err
	}

	return parseFloat64(&loaded.Protection.AdminRatePerSecond, envAdminRate)
}

func loadBooleans(loaded *Config) error {
	err := parseBool(&loaded.Protection.TrustProxyHeaders, envTrustProxy)
	if err != nil {
		return err
	}

	return parseBool(&loaded.CORS.AllowCredentials, envCORSAllowCredentials)
}

func loadLists(loaded *Config) {
	loaded.CORS.AllowedOrigins = parseList(envCORSAllowedOrigins, nil)
	loaded.CORS.AllowedMethods = parseList(envCORSAllowedMethods, loaded.CORS.AllowedMethods)
	loaded.CORS.AllowedHeaders = parseList(envCORSAllowedHeaders, loaded.CORS.AllowedHeaders)
	loaded.CORS.ExposedHeaders = parseList(envCORSExposedHeaders, loaded.CORS.ExposedHeaders)
}

func parseDuration(target *time.Duration, name string) error {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("%w: %s must be a duration: %w", errInvalidEnvironment, name, err)
	}

	*target = parsed

	return nil
}

func parseInt64(target *int64, name string) error {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %s must be an integer: %w", errInvalidEnvironment, name, err)
	}

	*target = parsed

	return nil
}

func parseInt(target *int, name string) error {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("%w: %s must be an integer: %w", errInvalidEnvironment, name, err)
	}

	*target = parsed

	return nil
}

func parseFloat64(target *float64, name string) error {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}

	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("%w: %s must be a number: %w", errInvalidEnvironment, name, err)
	}

	*target = parsed

	return nil
}

func parseBool(target *bool, name string) error {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("%w: %s must be a boolean: %w", errInvalidEnvironment, name, err)
	}

	*target = parsed

	return nil
}

func parseList(name string, fallback []string) []string {
	raw, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(raw) == "" {
		return clone(fallback)
	}

	parts := strings.Split(raw, ",")

	result := make([]string, 0, len(parts))

	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}

	return result
}
