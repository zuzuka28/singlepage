package page

import (
	"context"

	modelpage "singlepage/internal/model/page"
)

// Delete removes a page by its opaque identifier.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM pages WHERE id = ?`, id)
	if err != nil {
		return mapError(err)
	}

	return requireOneRow(result, modelpage.ErrNotFound)
}
