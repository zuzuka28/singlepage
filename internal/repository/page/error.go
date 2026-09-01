package page

import (
	"database/sql"
	"errors"
	"fmt"

	sqlite3 "github.com/ncruces/go-sqlite3"

	modelpage "singlepage/internal/model/page"
)

func requireOneRow(result sql.Result, zeroRowsError error) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected sqlite rows: %w", err)
	}

	if rows != 1 {
		return zeroRowsError
	}

	return nil
}

func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return modelpage.ErrNotFound
	}

	if errors.Is(err, sqlite3.FULL) {
		return modelpage.ErrQuotaExceeded
	}

	if errors.Is(err, sqlite3.CONSTRAINT_PRIMARYKEY) ||
		errors.Is(err, sqlite3.CONSTRAINT_UNIQUE) {
		return modelpage.ErrConflict
	}

	return fmt.Errorf("sqlite operation: %w", err)
}
