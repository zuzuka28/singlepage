package config

import (
	"fmt"
	"net"
)

const (
	envMetricsListen     = "SINGLEPAGE_METRICS_LISTEN"
	defaultMetricsListen = "127.0.0.1:9090"
)

// Metrics contains the private Prometheus server settings.
type Metrics struct {
	Listen string
}

func defaultMetrics() Metrics {
	return Metrics{Listen: defaultMetricsListen}
}

func loadMetrics(metrics *Metrics) {
	loadString(&metrics.Listen, envMetricsListen)
}

func validateMetrics(httpConfig HTTP, metrics Metrics) error {
	metricsHost, metricsPort, err := net.SplitHostPort(metrics.Listen)
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
