package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"singlepage/internal/config"
	"singlepage/internal/handler/httpapi"
	"singlepage/internal/metrics"
	"singlepage/internal/provider"
)

type serverResult struct {
	name string
	err  error
}

func run(ctx context.Context) error {
	applicationConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger := newLogger()
	applicationMetrics := metrics.New()

	publicServer, cleanup, err := provider.InitializeHTTPServer(
		ctx,
		applicationConfig,
		applicationMetrics,
		logger,
		false,
	)
	if err != nil {
		return fmt.Errorf("initialize daemon: %w", err)
	}
	defer cleanup()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", applicationMetrics.Handler())
	metricsServer := newMetricsServer(applicationConfig, metricsMux)

	return runServers(ctx, publicServer, metricsServer, applicationConfig.HTTP.ShutdownTimeout, logger)
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false, Level: slog.LevelInfo, ReplaceAttr: nil,
	}))
}

func newMetricsServer(applicationConfig config.Config, handler http.Handler) *http.Server {
	return &http.Server{ //nolint:exhaustruct // TLS and connection hooks use net/http defaults.
		Addr: applicationConfig.Metrics.Listen, Handler: handler,
		ReadHeaderTimeout: applicationConfig.HTTP.ReadHeaderTimeout,
		ReadTimeout:       applicationConfig.HTTP.ReadTimeout,
		WriteTimeout:      applicationConfig.HTTP.WriteTimeout,
		IdleTimeout:       applicationConfig.HTTP.IdleTimeout,
	}
}

func runServers(
	ctx context.Context,
	publicServer *httpapi.Server,
	metricsServer *http.Server,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	serveContext, cancelServe := context.WithCancel(ctx)
	defer cancelServe()

	results := make(chan serverResult, 2)
	go func() {
		results <- serverResult{name: "public", err: publicServer.Run(serveContext)}

		cancelServe()
	}()
	go func() {
		results <- serverResult{
			name: "metrics",
			err:  runMetricsServer(serveContext, metricsServer, shutdownTimeout, logger),
		}

		cancelServe()
	}()

	var resultErr error

	for range 2 {
		result := <-results
		if result.err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("run %s server: %w", result.name, result.err))
		}
	}

	return resultErr
}

func runMetricsServer(
	ctx context.Context,
	server *http.Server,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	logger.InfoContext(ctx, "metrics HTTP server listening", "address", server.Addr)

	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.ListenAndServe()
	}()

	select {
	case err := <-serveResult:
		return normalizeMetricsServeError(err)

	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		shutdownErr := server.Shutdown(shutdownContext)
		serveErr := <-serveResult

		if shutdownErr != nil {
			shutdownErr = fmt.Errorf("shutdown metrics HTTP: %w", shutdownErr)
		}

		return errors.Join(normalizeMetricsServeError(serveErr), shutdownErr)
	}
}

func normalizeMetricsServeError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("serve metrics HTTP: %w", err)
}
