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
	listen := flag.String("listen", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "data.db", "SQLite database path")
	flag.Parse()

	frontend, err := fs.Sub(frontendFiles, "web/dist")
	if err != nil {
		log.Fatal(err)
	}
	fallback := []byte(`<!doctype html><html><body><main><h1>Frontend is not built</h1><p>Run npm run build and restart the server.</p></main></body></html>`)
	app, err := server.Open(context.Background(), *dbPath, frontend, fallback)
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
