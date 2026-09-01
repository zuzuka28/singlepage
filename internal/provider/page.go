package provider

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"singlepage/internal/config"
	repositorypage "singlepage/internal/repository/page"
	servicepage "singlepage/internal/service/page"
)

// ProvideStorage extracts the transport-independent storage block.
func ProvideStorage(applicationConfig config.Config) config.Storage {
	return applicationConfig.Storage()
}

// ProvidePageRepository opens the page persistence adapter.
func ProvidePageRepository(
	ctx context.Context,
	storage config.Storage,
	logger *slog.Logger,
) (*repositorypage.Repository, func(), error) {
	repository, err := repositorypage.Open(ctx, storage.SQLite.DSN, storage.SQLite.MaxBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("open page repository: %w", err)
	}

	var once sync.Once

	cleanup := func() {
		once.Do(func() {
			closeErr := repository.Close()
			if closeErr != nil {
				logger.Error("close page repository", "error", closeErr)
			}
		})
	}

	return repository, cleanup, nil
}

// ProvidePageService creates the page domain service.
func ProvidePageService(
	repository *repositorypage.Repository,
	storage config.Storage,
) *servicepage.Service {
	return servicepage.New(repository, servicepage.Config{MaxPages: storage.Page.MaxPages})
}
