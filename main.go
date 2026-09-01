package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"singlepage/internal/server"
)

// Embedding web (rather than only web/dist) keeps development builds valid even
// before Vite has produced dist. Only the dist subdirectory is ever served.
//
//go:embed all:web
var frontendFiles embed.FS

func main() {
	config := server.DefaultConfig()
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	dbPath := flag.String("db", "data.db", "SQLite database path")
	adminTokenFile := flag.String("admin-token-file", "", "optional file containing the admin deletion token")
	flag.Int64Var(&config.MaxRequestBodyBytes, "max-request-bytes", config.MaxRequestBodyBytes, "maximum JSON request body size")
	flag.IntVar(&config.MaxCiphertextBytes, "max-page-bytes", config.MaxCiphertextBytes, "maximum encrypted page size")
	flag.Int64Var(&config.MaxDatabaseBytes, "max-database-bytes", config.MaxDatabaseBytes, "maximum SQLite logical size; 0 disables")
	flag.Int64Var(&config.MaxPages, "max-pages", config.MaxPages, "maximum number of stored pages; 0 disables")
	flag.Float64Var(&config.CreateRatePerSecond, "create-rate", config.CreateRatePerSecond, "per-client sustained page creations per second; 0 disables")
	flag.IntVar(&config.CreateBurst, "create-burst", config.CreateBurst, "per-client page creation burst")
	flag.BoolVar(&config.TrustProxyHeaders, "trust-proxy-headers", false, "use the last X-Forwarded-For address for rate limiting")
	flag.Parse()
	if *adminTokenFile != "" {
		raw, err := os.ReadFile(*adminTokenFile)
		if err != nil {
			log.Fatal(fmt.Errorf("read admin token: %w", err))
		}
		config.AdminToken = strings.TrimSpace(string(raw))
		if config.AdminToken == "" {
			log.Fatal("admin token file is empty")
		}
	}

	frontend, err := fs.Sub(frontendFiles, "web/dist")
	if err != nil {
		log.Fatal(err)
	}
	fallback := []byte(`<!doctype html><html><body><main><h1>Frontend is not built</h1><p>Run npm run build and restart the server.</p></main></body></html>`)
	app, err := server.OpenWithConfig(context.Background(), *dbPath, frontend, fallback, config)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", *listen)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(fmt.Errorf("serve: %w", err))
	}
}
