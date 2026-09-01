// Package metrics exposes HTTP API usage metrics in the Prometheus format.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics owns an isolated Prometheus registry and its HTTP collectors.
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	errors   *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}
