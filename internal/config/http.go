package config

import (
	"strings"
	"time"
)

const (
	envHTTPListen            = "SINGLEPAGE_HTTP_LISTEN"
	envHTTPReadHeaderTimeout = "SINGLEPAGE_HTTP_READ_HEADER_TIMEOUT"
	envHTTPReadTimeout       = "SINGLEPAGE_HTTP_READ_TIMEOUT"
	envHTTPWriteTimeout      = "SINGLEPAGE_HTTP_WRITE_TIMEOUT"
	envHTTPIdleTimeout       = "SINGLEPAGE_HTTP_IDLE_TIMEOUT"
	envHTTPShutdownTimeout   = "SINGLEPAGE_HTTP_SHUTDOWN_TIMEOUT"

	defaultListen            = "127.0.0.1:8080"
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 20 * time.Second
	defaultWriteTimeout      = 20 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

// HTTP contains HTTP server lifecycle settings.
type HTTP struct {
	Listen            string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func defaultHTTP() HTTP {
	return HTTP{
		Listen:            defaultListen,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ShutdownTimeout:   defaultShutdownTimeout,
	}
}

func loadHTTP(httpConfig *HTTP) error {
	loadString(&httpConfig.Listen, envHTTPListen)

	values := []struct {
		name   string
		target *time.Duration
	}{
		{name: envHTTPReadHeaderTimeout, target: &httpConfig.ReadHeaderTimeout},
		{name: envHTTPReadTimeout, target: &httpConfig.ReadTimeout},
		{name: envHTTPWriteTimeout, target: &httpConfig.WriteTimeout},
		{name: envHTTPIdleTimeout, target: &httpConfig.IdleTimeout},
		{name: envHTTPShutdownTimeout, target: &httpConfig.ShutdownTimeout},
	}

	for _, value := range values {
		err := parseDuration(value.target, value.name)
		if err != nil {
			return err
		}
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
