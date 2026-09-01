package page

import "fmt"

// Close closes the repository database connection.
func (r *Repository) Close() error {
	err := r.db.Close()
	if err != nil {
		return fmt.Errorf("close sqlite: %w", err)
	}

	return nil
}
