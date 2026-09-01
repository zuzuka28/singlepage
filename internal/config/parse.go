package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func loadString(target *string, name string) {
	if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
		*target = raw
	}
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
