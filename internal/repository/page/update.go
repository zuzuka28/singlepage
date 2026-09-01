package page

import (
	"context"

	modelpage "singlepage/internal/model/page"
)

// Update replaces page data when its revision and write capability still match.
func (r *Repository) Update(ctx context.Context, cmd modelpage.UpdateRepositoryCmd) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE pages
		SET revision = revision + 1, salt = ?, write_token_hash = ?, ciphertext = ?, updated_at = ?
		WHERE id = ? AND revision = ? AND write_token_hash = ?`,
		cmd.Salt,
		cmd.WriteTokenHash,
		cmd.Ciphertext,
		cmd.UpdatedAt.Unix(),
		cmd.ID,
		cmd.ExpectedRevision,
		cmd.ExpectedWriteTokenHash,
	)
	if err != nil {
		return mapError(err)
	}

	return requireOneRow(result, modelpage.ErrConflict)
}
