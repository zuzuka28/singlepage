package page

import (
	"context"

	modelpage "singlepage/internal/model/page"
)

// Rotate atomically replaces a page identifier and write capability.
func (r *Repository) Rotate(ctx context.Context, cmd modelpage.RotateRepositoryCmd) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE pages
		SET id = ?, revision = 1, salt = ?, write_token_hash = ?, ciphertext = ?, updated_at = ?
		WHERE id = ? AND revision = ? AND write_token_hash = ?`,
		cmd.NewID,
		cmd.Salt,
		cmd.NewWriteTokenHash,
		cmd.Ciphertext,
		cmd.UpdatedAt.Unix(),
		cmd.OldID,
		cmd.ExpectedRevision,
		cmd.ExpectedWriteTokenHash,
	)
	if err != nil {
		return mapError(err)
	}

	return requireOneRow(result, modelpage.ErrConcurrentChange)
}
