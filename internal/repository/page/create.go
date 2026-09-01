package page

import (
	"context"
	"database/sql"
	"fmt"

	modelpage "singlepage/internal/model/page"
)

// Create persists a new page if the configured page quota permits it.
func (r *Repository) Create(ctx context.Context, cmd modelpage.CreateRepositoryCmd) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mapError(err)
	}

	defer func() { _ = tx.Rollback() }()

	if cmd.MaxPages > 0 {
		var count int64

		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM pages`).Scan(&count)
		if err != nil {
			return mapError(err)
		}

		if count >= cmd.MaxPages {
			return modelpage.ErrQuotaExceeded
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO pages (id, revision, salt, write_token_hash, ciphertext, updated_at)
		VALUES (?, 1, ?, ?, ?, ?)`,
		cmd.ID,
		cmd.Salt,
		cmd.WriteTokenHash,
		cmd.Ciphertext,
		cmd.UpdatedAt.Unix(),
	)
	if err != nil {
		return mapError(err)
	}

	return commitCreate(tx)
}

func commitCreate(tx *sql.Tx) error {
	err := tx.Commit()
	if err != nil {
		return fmt.Errorf("commit create page: %w", mapError(err))
	}

	return nil
}
