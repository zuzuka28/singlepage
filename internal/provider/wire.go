//go:build wireinject

package provider

import (
	"context"
	"log/slog"

	"github.com/google/wire"

	"singlepage/internal/config"
	"singlepage/internal/handler/httpapi"
	"singlepage/internal/metrics"
	servicepage "singlepage/internal/service/page"
)

// InitializePageService builds the transport-independent page graph.
func InitializePageService(
	ctx context.Context,
	storage config.Storage,
	logger *slog.Logger,
) (*servicepage.Service, func(), error) {
	wire.Build(
		ProvidePageRepository,
		ProvidePageService,
	)

	return nil, nil, nil
}

// InitializeHTTPServer builds the public HTTP server dependency graph.
// The page providers intentionally remain explicit in both roots: a shared
// package-level wire.NewSet is copied into wire_gen.go and makes production
// binaries import Wire at runtime.
func InitializeHTTPServer(
	ctx context.Context,
	applicationConfig config.Config,
	applicationMetrics *metrics.Metrics,
	logger *slog.Logger,
	includeFrontend bool,
) (*httpapi.Server, func(), error) {
	wire.Build(
		ProvideStorage,
		ProvidePageRepository,
		ProvidePageService,
		ProvideHTTPServer,
	)

	return nil, nil, nil
}
