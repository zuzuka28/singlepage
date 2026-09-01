package config

import (
	"fmt"
	"math"
	"os"
	"strings"
)

const (
	envRequestMaxBody = "SINGLEPAGE_REQUEST_MAX_BODY_BYTES"
	envCreateRate     = "SINGLEPAGE_CREATE_RATE_PER_SECOND"
	envCreateBurst    = "SINGLEPAGE_CREATE_BURST"
	envAdminRate      = "SINGLEPAGE_ADMIN_RATE_PER_SECOND"
	envAdminBurst     = "SINGLEPAGE_ADMIN_BURST"
	envTrustProxy     = "SINGLEPAGE_TRUST_PROXY_HEADERS"
	envAdminTokenFile = "SINGLEPAGE_ADMIN_TOKEN_FILE"

	defaultMaxRequestBody = 16 << 20
	defaultCreateRate     = 1
	defaultCreateBurst    = 20
	defaultAdminRate      = 1
	defaultAdminBurst     = 5
	maxAdminTokenBytes    = 256
	minAdminTokenBytes    = 32
)

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

func defaultProtection() Protection {
	return Protection{
		MaxRequestBodyBytes: defaultMaxRequestBody,
		CreateRatePerSecond: defaultCreateRate,
		CreateBurst:         defaultCreateBurst,
		AdminRatePerSecond:  defaultAdminRate,
		AdminBurst:          defaultAdminBurst,
		TrustProxyHeaders:   false,
		AdminToken:          "",
	}
}

func loadProtection(protection *Protection) error {
	err := parseInt64(&protection.MaxRequestBodyBytes, envRequestMaxBody)
	if err != nil {
		return err
	}

	err = parseFloat64(&protection.CreateRatePerSecond, envCreateRate)
	if err != nil {
		return err
	}

	err = parseInt(&protection.CreateBurst, envCreateBurst)
	if err != nil {
		return err
	}

	err = parseFloat64(&protection.AdminRatePerSecond, envAdminRate)
	if err != nil {
		return err
	}

	err = parseInt(&protection.AdminBurst, envAdminBurst)
	if err != nil {
		return err
	}

	err = parseBool(&protection.TrustProxyHeaders, envTrustProxy)
	if err != nil {
		return err
	}

	return loadAdminToken(protection)
}

func loadAdminToken(protection *Protection) error {
	path := strings.TrimSpace(os.Getenv(envAdminTokenFile))
	if path == "" {
		return nil
	}

	raw, err := os.ReadFile(path) // #nosec G703 -- The operator explicitly configures this secret file.
	if err != nil {
		return fmt.Errorf("%w: read %s: %w", errInvalidEnvironment, envAdminTokenFile, err)
	}

	token := strings.TrimSpace(string(raw))
	if token == "" {
		return invalid(envAdminTokenFile, "token file must not be empty")
	}

	protection.AdminToken = token

	return nil
}

func validateProtection(protection Protection) error {
	if protection.MaxRequestBodyBytes < 1 {
		return invalid(envRequestMaxBody, "must be positive")
	}

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
	if rate < 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
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
