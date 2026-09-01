package provider_test

import (
	"log/slog"
	"path/filepath"
	"testing"

	"singlepage/internal/config"
	"singlepage/internal/provider"
)

func TestPageServiceCleanupIsIdempotent(t *testing.T) {
	t.Parallel()

	_, cleanup, err := provider.InitializePageService(
		t.Context(),
		config.Storage{
			SQLite: config.SQLite{DSN: filepath.Join(t.TempDir(), "data.db"), MaxBytes: 1 << 20},
			Page:   config.Page{MaxPages: 10},
		},
		slog.New(slog.DiscardHandler),
	)
	if err != nil {
		t.Fatalf("InitializePageService: %v", err)
	}

	cleanup()
	cleanup()
}
