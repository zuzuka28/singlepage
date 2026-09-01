package page

import (
	"database/sql"
	"errors"
	"fmt"

	sqlite3 "github.com/mattn/go-sqlite3"

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

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if sqliteErr.Code == sqlite3.ErrFull {
			return modelpage.ErrQuotaExceeded
		}

		if sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey ||
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return modelpage.ErrConflict
		}
	}

	return fmt.Errorf("sqlite operation: %w", err)
}
