package page

import (
	"context"
	"database/sql"
	"fmt"
)

// Open opens and initializes a SQLite page repository.
func Open(ctx context.Context, dsn string, maxDatabaseBytes int64) (*Repository, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	repository := &Repository{db: db}

	err = repository.initialize(ctx, maxDatabaseBytes)
	if err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("initialize repository: %w", err)
	}

	return repository, nil
}
