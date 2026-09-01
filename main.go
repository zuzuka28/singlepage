package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"singlepage/internal/config"
	"singlepage/internal/handler/httpapi"
	"singlepage/internal/metrics"
	repositorypage "singlepage/internal/repository/page"
	servicepage "singlepage/internal/service/page"
)

type serverBinding struct {
	name   string
	server *http.Server
}

type serverResult struct {
	name string
	err  error
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx)

	stop()

	if err != nil {
		slog.Error("application stopped", "error", err)

		os.Exit(1)
	}
}

func run(ctx context.Context) (returnErr error) {
	applicationConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	repository, err := repositorypage.Open(
		ctx,
		applicationConfig.SQLite.DSN,
		applicationConfig.SQLite.MaxBytes,
	)
	if err != nil {
		return fmt.Errorf("open page repository: %w", err)
	}

	defer func() {
		returnErr = errors.Join(returnErr, repository.Close())
	}()

	pageService := servicepage.New(repository, servicepage.Config{
		MaxPages: applicationConfig.Page.MaxPages,
	})
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelInfo,
		ReplaceAttr: nil,
	}))
	applicationMetrics := metrics.New()
	applicationHandler := httpapi.New(pageService, applicationConfig, applicationMetrics, logger)
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", applicationMetrics.Handler())
	servers := []serverBinding{
		{
			name: "application",
			server: newHTTPServer(
				applicationConfig.HTTP.Listen,
				applicationConfig.HTTP,
				applicationHandler,
			),
		},
		{
			name: "metrics",
			server: newHTTPServer(
				applicationConfig.Metrics.Listen,
				applicationConfig.HTTP,
				metricsMux,
			),
		},
	}

	return serve(ctx, servers, applicationConfig.HTTP.ShutdownTimeout, logger)
}

func newHTTPServer(listen string, httpConfig config.HTTP, handler http.Handler) *http.Server {
	return &http.Server{ //nolint:exhaustruct // TLS and connection hooks intentionally use net/http defaults.
		Addr:              listen,
		Handler:           handler,
		ReadHeaderTimeout: httpConfig.ReadHeaderTimeout,
		ReadTimeout:       httpConfig.ReadTimeout,
		WriteTimeout:      httpConfig.WriteTimeout,
		IdleTimeout:       httpConfig.IdleTimeout,
	}
}

func serve(
	ctx context.Context,
	servers []serverBinding,
	shutdownTimeout time.Duration,
	logger *slog.Logger,
) error {
	serveContext, cancelServe := context.WithCancel(ctx)
	defer cancelServe()

	results := make(chan serverResult, len(servers))
	for _, binding := range servers {
		logger.InfoContext(ctx, binding.name+" server listening", "address", binding.server.Addr)

		go func(current serverBinding) {
			results <- serverResult{name: current.name, err: current.server.ListenAndServe()}

			cancelServe()
		}(binding)
	}

	shutdownResult := make(chan error, 1)
	go shutdownServers(serveContext, servers, shutdownTimeout, shutdownResult)

	var serveErr error

	for range servers {
		result := <-results
		if result.err != nil && !errors.Is(result.err, http.ErrServerClosed) {
			serveErr = errors.Join(serveErr, fmt.Errorf("serve %s HTTP: %w", result.name, result.err))
		}
	}

	return errors.Join(serveErr, <-shutdownResult)
}

func shutdownServers(
	ctx context.Context,
	servers []serverBinding,
	shutdownTimeout time.Duration,
	result chan<- error,
) {
	<-ctx.Done()

	shutdownContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	var shutdownErr error

	for _, binding := range servers {
		err := binding.server.Shutdown(shutdownContext)
		if err != nil {
			shutdownErr = errors.Join(
				shutdownErr,
				fmt.Errorf("shutdown %s HTTP: %w", binding.name, err),
			)
		}
	}

	result <- shutdownErr
}
