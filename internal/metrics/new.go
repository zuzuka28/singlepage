package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	metricNamespace = "singlepage"
	metricSubsystem = "api"
)

// New constructs metrics backed by a private Prometheus registry.
func New() *Metrics {
	var requestsOptions prometheus.CounterOpts

	requestsOptions.Namespace = metricNamespace
	requestsOptions.Subsystem = metricSubsystem
	requestsOptions.Name = "requests_total"
	requestsOptions.Help = "Total number of API requests."

	var errorsOptions prometheus.CounterOpts

	errorsOptions.Namespace = metricNamespace
	errorsOptions.Subsystem = metricSubsystem
	errorsOptions.Name = "request_errors_total"
	errorsOptions.Help = "Total number of API requests completed with an error status."

	var durationOptions prometheus.HistogramOpts

	durationOptions.Namespace = metricNamespace
	durationOptions.Subsystem = metricSubsystem
	durationOptions.Name = "request_duration_seconds"
	durationOptions.Help = "API request duration in seconds."
	durationOptions.Buckets = prometheus.DefBuckets

	var inFlightOptions prometheus.GaugeOpts

	inFlightOptions.Namespace = metricNamespace
	inFlightOptions.Subsystem = metricSubsystem
	inFlightOptions.Name = "requests_in_flight"
	inFlightOptions.Help = "Current number of API requests being served."

	labels := []string{"method", "route", "status"}
	result := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(requestsOptions, labels),
		errors:   prometheus.NewCounterVec(errorsOptions, labels),
		duration: prometheus.NewHistogramVec(durationOptions, labels),
		inFlight: prometheus.NewGauge(inFlightOptions),
	}
	result.registry.MustRegister(result.requests, result.errors, result.duration, result.inFlight)

	return result
}
