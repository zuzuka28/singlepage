//go:build wails

package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"singlepage/cmd/app/internal/session"
	"singlepage/internal/config"
	"singlepage/internal/provider"
	servicepage "singlepage/internal/service/page"
)

type pageDependencies struct {
	pages    *servicepage.Service
	locators *session.Store
	cleanup  func()
}

func openPageDependencies(ctx context.Context, logger *slog.Logger) (pageDependencies, error) {
	storage, err := config.LoadStorage()
	if err != nil {
		return pageDependencies{}, fmt.Errorf("load native storage configuration: %w", err)
	}

	dataDirectory, err := session.DataDir()
	if err != nil {
		return pageDependencies{}, fmt.Errorf("resolve native data directory: %w", err)
	}

	err = os.MkdirAll(dataDirectory, 0o700)
	if err != nil {
		return pageDependencies{}, fmt.Errorf("create native data directory: %w", err)
	}

	storage.SQLite.DSN = filepath.Join(dataDirectory, "data.db")

	pages, cleanup, err := provider.InitializePageService(ctx, storage, logger)
	if err != nil {
		return pageDependencies{}, fmt.Errorf("open native page service: %w", err)
	}

	return pageDependencies{
		pages: pages, locators: session.New(dataDirectory), cleanup: cleanup,
	}, nil
}
