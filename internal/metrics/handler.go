package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler exposes the private registry in the Prometheus text format.
func (metrics *Metrics) Handler() http.Handler {
	var options promhttp.HandlerOpts

	options.Registry = metrics.registry

	return promhttp.HandlerFor(metrics.registry, options)
}
