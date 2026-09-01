package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"singlepage/internal/config"
	"singlepage/internal/metrics"
)

// Server owns the public HTTP listener and its graceful-shutdown lifecycle.
// The private metrics HTTP server remains a process-entrypoint responsibility.
type Server struct {
	server          *http.Server
	shutdownTimeout time.Duration
	logger          *slog.Logger
}

// New creates the public HTTP server. includeFrontend selects the browser or
// headless API surface without changing the API contract.
func New(
	pages pageService,
	applicationConfig config.Config,
	applicationMetrics *metrics.Metrics,
	logger *slog.Logger,
	includeFrontend bool,
) *Server {
	handler := newHandler(
		pages,
		applicationConfig,
		applicationMetrics,
		logger,
		includeFrontend,
	)

	return &Server{
		server:          newHTTPServer(applicationConfig.HTTP.Listen, applicationConfig.HTTP, handler),
		shutdownTimeout: applicationConfig.HTTP.ShutdownTimeout,
		logger:          logger,
	}
}

// ServeHTTP delegates to the configured public handler for transport tests and embedding.
func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	server.server.Handler.ServeHTTP(response, request)
}

// Run serves the public listener until it fails or the context is cancelled.
func (server *Server) Run(ctx context.Context) error {
	server.logger.InfoContext(ctx, "public HTTP server listening", "address", server.server.Addr)

	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.server.ListenAndServe()
	}()

	select {
	case err := <-serveResult:
		return normalizeServeError(err)

	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			server.shutdownTimeout,
		)
		defer cancel()

		shutdownErr := server.server.Shutdown(shutdownContext)
		serveErr := <-serveResult

		if shutdownErr != nil {
			shutdownErr = fmt.Errorf("shutdown public HTTP: %w", shutdownErr)
		}

		return errors.Join(normalizeServeError(serveErr), shutdownErr)
	}
}

func newHTTPServer(listen string, httpConfig config.HTTP, handler http.Handler) *http.Server {
	return &http.Server{ //nolint:exhaustruct // TLS and connection hooks use net/http defaults.
		Addr: listen, Handler: handler,
		ReadHeaderTimeout: httpConfig.ReadHeaderTimeout,
		ReadTimeout:       httpConfig.ReadTimeout,
		WriteTimeout:      httpConfig.WriteTimeout,
		IdleTimeout:       httpConfig.IdleTimeout,
	}
}

func normalizeServeError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("serve public HTTP: %w", err)
}
