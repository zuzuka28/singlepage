package page //nolint:testpackage // The test verifies private connection initialization.

import (
	"path/filepath"
	"testing"
)

func TestOpenAppliesConnectionPragmas(t *testing.T) {
	t.Parallel()

	repository, err := Open(t.Context(), filepath.Join(t.TempDir(), "data.db"), 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	t.Cleanup(func() {
		closeErr := repository.Close()
		if closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	var journalMode string

	err = repository.db.QueryRowContext(t.Context(), `PRAGMA journal_mode`).Scan(&journalMode)
	if err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}

	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	var busyTimeout int

	err = repository.db.QueryRowContext(t.Context(), `PRAGMA busy_timeout`).Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}

	if busyTimeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busyTimeout)
	}
}
