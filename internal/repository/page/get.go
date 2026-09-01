package page

import (
	"context"
	"time"

	modelpage "singlepage/internal/model/page"
)

// Get loads a page by its opaque identifier.
func (r *Repository) Get(ctx context.Context, id string) (modelpage.RepositoryPage, error) {
	var page modelpage.RepositoryPage

	var updatedAt int64

	err := r.db.QueryRowContext(ctx, `
		SELECT id, revision, salt, write_token_hash, ciphertext, updated_at
		FROM pages WHERE id = ?`, id,
	).Scan(
		&page.ID,
		&page.Revision,
		&page.Salt,
		&page.WriteTokenHash,
		&page.Ciphertext,
		&updatedAt,
	)
	if err != nil {
		return modelpage.RepositoryPage{}, mapError(err)
	}

	page.UpdatedAt = time.Unix(updatedAt, 0)

	return page, nil
}
