//go:build wails

package session_test

import (
	"os"
	"testing"

	"singlepage/cmd/app/internal/session"
)

func TestStoreRoundTrip(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	store := session.New(directory)
	current := "/p/0123456789abcdef#0123456789abcdef"
	previous := "/p/fedcba9876543210#fedcba9876543210"

	err := store.Write(current, previous)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	gotCurrent, gotPrevious, err := store.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if gotCurrent != current || gotPrevious != previous {
		t.Fatalf("Read() = %q, %q", gotCurrent, gotPrevious)
	}

	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("ReadDir() = %v, %v", entries, err)
	}

	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("locator mode = %o, want 600", info.Mode().Perm())
	}
}

func TestStoreDefaultsToRoot(t *testing.T) {
	t.Parallel()

	current, previous, err := session.New(t.TempDir()).Read()
	if err != nil || current != "/" || previous != "" {
		t.Fatalf("Read() = %q, %q, %v", current, previous, err)
	}
}
