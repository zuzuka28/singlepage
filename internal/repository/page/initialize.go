package page

import (
	"context"
	"errors"
	"fmt"
)

var (
	errNegativeDatabaseSize = errors.New("database size limit cannot be negative")
	errDatabaseTooLarge     = errors.New("sqlite database already exceeds configured storage limit")
	errDatabaseLimitRefused = errors.New("sqlite refused configured storage limit")
)

func (r *Repository) initialize(ctx context.Context, maxDatabaseBytes int64) error {
	_, err := r.db.ExecContext(ctx, `
		PRAGMA journal_mode=WAL;
		PRAGMA busy_timeout=5000;
		CREATE TABLE IF NOT EXISTS pages (
			id TEXT PRIMARY KEY,
			revision INTEGER NOT NULL CHECK (revision >= 1),
			salt BLOB NOT NULL,
			write_token_hash BLOB NOT NULL CHECK (length(write_token_hash) = 32),
			ciphertext BLOB NOT NULL,
			updated_at INTEGER NOT NULL
		);`)
	if err != nil {
		return fmt.Errorf("initialize sqlite schema: %w", err)
	}

	if maxDatabaseBytes == 0 {
		return nil
	}

	if maxDatabaseBytes < 0 {
		return errNegativeDatabaseSize
	}

	return r.applyDatabaseSizeLimit(ctx, maxDatabaseBytes)
}

func (r *Repository) applyDatabaseSizeLimit(ctx context.Context, maxDatabaseBytes int64) error {
	var pageSize int64

	err := r.db.QueryRowContext(ctx, `PRAGMA page_size`).Scan(&pageSize)
	if err != nil {
		return fmt.Errorf("read sqlite page size: %w", err)
	}

	var pageCount int64

	err = r.db.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&pageCount)
	if err != nil {
		return fmt.Errorf("read sqlite page count: %w", err)
	}

	maxPageCount := maxDatabaseBytes / pageSize
	if maxPageCount < pageCount {
		return errDatabaseTooLarge
	}

	query := fmt.Sprintf(`PRAGMA max_page_count = %d`, maxPageCount)

	var applied int64

	err = r.db.QueryRowContext(ctx, query).Scan(&applied)
	if err != nil {
		return fmt.Errorf("limit sqlite database size: %w", err)
	}

	if applied > maxPageCount {
		return errDatabaseLimitRefused
	}

	return nil
}
